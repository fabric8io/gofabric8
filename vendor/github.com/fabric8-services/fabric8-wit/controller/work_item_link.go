package controller

import (
	"context"
	"reflect"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkController implements the work-item-link resource.
type WorkItemLinkController struct {
	*goa.Controller
	db     application.DB
	config WorkItemLinkControllerConfig
}

// WorkItemLinkControllerConfig the config interface for the WorkitemLinkController
type WorkItemLinkControllerConfig interface {
	GetCacheControlWorkItemLinks() string
}

// NewWorkItemLinkController creates a work-item-link controller.
func NewWorkItemLinkController(service *goa.Service, db application.DB, config WorkItemLinkControllerConfig) *WorkItemLinkController {
	return &WorkItemLinkController{
		Controller: service.NewController("WorkItemLinkController"),
		db:         db,
		config:     config,
	}
}

// Instead of using app.WorkItemLinkHref directly, we store a function object
// inside the WorkItemLinkContext in order to generate proper links nomatter
// from which controller the context is being used. For example one could use
// app.WorkItemRelationshipsLinksHref.
type hrefLinkFunc func(obj interface{}) string

// workItemLinkContext bundles objects that are needed by most of the functions.
// It can easily be extended.
type workItemLinkContext struct {
	RequestData           *goa.RequestData
	ResponseData          *goa.ResponseData
	Application           application.Application
	Context               context.Context
	CurrentUserIdentityID *uuid.UUID
	DB                    application.DB
	LinkFunc              hrefLinkFunc
}

// newWorkItemLinkContext returns a new workItemLinkContext
func newWorkItemLinkContext(ctx context.Context, appl application.Application, db application.DB, requestData *goa.RequestData, responseData *goa.ResponseData, linkFunc hrefLinkFunc, currentUserIdentityID *uuid.UUID) *workItemLinkContext {
	return &workItemLinkContext{
		RequestData:           requestData,
		ResponseData:          responseData,
		Application:           appl,
		Context:               ctx,
		CurrentUserIdentityID: currentUserIdentityID,
		DB:       db,
		LinkFunc: linkFunc,
	}
}

// getTypesOfLinks returns an array of distinct work item link types for the
// given work item links
func getTypesOfLinks(ctx *workItemLinkContext, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItemLinkTypeData, error) {
	// Build our "set" of distinct type IDs already converted as strings
	typeIDMap := map[uuid.UUID]bool{}
	for _, linkData := range linksDataArr {
		typeIDMap[linkData.Relationships.LinkType.Data.ID] = true
	}
	// Now include the optional link type data in the work item link "included" array
	linkTypeModels := []link.WorkItemLinkType{}
	for typeID := range typeIDMap {
		linkTypeModel, err := ctx.Application.WorkItemLinkTypes().Load(ctx.Context, typeID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		linkTypeModels = append(linkTypeModels, *linkTypeModel)
	}
	appLinkTypes, err := ConvertLinkTypesFromModels(ctx.RequestData, linkTypeModels)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return appLinkTypes.Data, nil
}

// getWorkItemsOfLinks returns an array of distinct work items as they appear as
// source or target in the given work item links.
func getWorkItemsOfLinks(ctx *workItemLinkContext, linksDataArr []*app.WorkItemLinkData) ([]*app.WorkItem, error) {
	// Build our "set" of distinct work item IDs already converted as strings
	workItemIDMap := map[uuid.UUID]bool{}
	for _, linkData := range linksDataArr {
		workItemIDMap[linkData.Relationships.Source.Data.ID] = true
		workItemIDMap[linkData.Relationships.Target.Data.ID] = true
	}
	// Now include the optional work item data in the work item link "included" array
	workItemArr := []*app.WorkItem{}
	for workItemID := range workItemIDMap {
		wi, err := ctx.Application.WorkItems().LoadByID(ctx.Context, workItemID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		workItemArr = append(workItemArr, ConvertWorkItem(ctx.RequestData, *wi))
	}
	return workItemArr, nil
}

// getCategoriesOfLinkTypes returns an array of distinct work item link
// categories for the given work item link types
func getCategoriesOfLinkTypes(ctx *workItemLinkContext, linkTypeDataArr []*app.WorkItemLinkTypeData) ([]*app.WorkItemLinkCategoryData, error) {
	// Build our "set" of distinct category IDs already converted as strings
	catIDMap := map[uuid.UUID]bool{}
	for _, linkTypeData := range linkTypeDataArr {
		catIDMap[linkTypeData.Relationships.LinkCategory.Data.ID] = true
	}
	// Now include the optional link category data in the work item link "included" array
	catDataArr := []*app.WorkItemLinkCategoryData{}
	for catID := range catIDMap {
		modelCategory, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, catID)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		appCategory := ConvertLinkCategoryFromModel(*modelCategory)
		catDataArr = append(catDataArr, appCategory.Data)
	}
	return catDataArr, nil
}

// enrichLinkSingle includes related resources in the link's "included" array
func enrichLinkSingle(ctx *workItemLinkContext, appLinks *app.WorkItemLinkSingle) error {
	// include link type
	modelLinkType, err := ctx.Application.WorkItemLinkTypes().Load(ctx.Context, appLinks.Data.Relationships.LinkType.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinkType := ConvertWorkItemLinkTypeFromModel(ctx.RequestData, *modelLinkType)
	appLinks.Included = append(appLinks.Included, appLinkType.Data)

	// include link category
	modelCategory, err := ctx.Application.WorkItemLinkCategories().Load(ctx.Context, appLinkType.Data.Relationships.LinkCategory.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appCategory := ConvertLinkCategoryFromModel(*modelCategory)
	appLinks.Included = append(appLinks.Included, appCategory.Data)

	// TODO(kwk): include source work item type (once #559 is merged)
	// sourceWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.SourceType.Data.ID)
	// if err != nil {
	// 	return errs.WithStack(err)
	// }
	// link.Included = append(link.Included, sourceWit.Data)

	// TODO(kwk): include target work item type (once #559 is merged)
	// targetWit, err := appl.WorkItemTypes().Load(ctx, linkType.Data.Relationships.TargetType.Data.ID)
	// if err != nil {
	// 	return errs.WithStack(err)
	// }
	// link.Included = append(link.Included, targetWit.Data)

	// TODO(kwk): include source work item
	sourceWi, err := ctx.Application.WorkItems().LoadByID(ctx.Context, appLinks.Data.Relationships.Source.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinks.Included = append(appLinks.Included, ConvertWorkItem(ctx.RequestData, *sourceWi))

	// TODO(kwk): include target work item
	targetWi, err := ctx.Application.WorkItems().LoadByID(ctx.Context, appLinks.Data.Relationships.Target.Data.ID)
	if err != nil {
		return errs.WithStack(err)
	}
	appLinks.Included = append(appLinks.Included, ConvertWorkItem(ctx.RequestData, *targetWi))

	// Add links to individual link data element
	relatedURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*appLinks.Data.ID))
	appLinks.Data.Links = &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}

	return nil
}

// enrichLinkList includes related resources in the linkArr's "included" element
func enrichLinkList(ctx *workItemLinkContext, linkArr *app.WorkItemLinkList) error {

	// include link types
	typeDataArr, err := getTypesOfLinks(ctx, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr := make([]interface{}, len(typeDataArr))
	for i, v := range typeDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// include link categories
	catDataArr, err := getCategoriesOfLinkTypes(ctx, typeDataArr)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(catDataArr))
	for i, v := range catDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WIs from source and target
	workItemDataArr, err := getWorkItemsOfLinks(ctx, linkArr.Data)
	if err != nil {
		return errs.WithStack(err)
	}
	// Convert slice of objects to slice of interface (see https://golang.org/doc/faq#convert_slice_of_interface)
	interfaceArr = make([]interface{}, len(workItemDataArr))
	for i, v := range workItemDataArr {
		interfaceArr[i] = v
	}
	linkArr.Included = append(linkArr.Included, interfaceArr...)

	// TODO(kwk): Include WITs (once #559 is merged)

	// Add links to individual link data element
	for _, link := range linkArr.Data {
		relatedURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*link.ID))
		link.Links = &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		}
	}
	return nil
}

type createWorkItemLinkFuncs interface {
	BadRequest(r *app.JSONAPIErrors) error
	Created(r *app.WorkItemLinkSingle) error
	InternalServerError(r *app.JSONAPIErrors) error
	Unauthorized(r *app.JSONAPIErrors) error
}

func createWorkItemLink(ctx *workItemLinkContext, httpFuncs createWorkItemLinkFuncs, payload *app.CreateWorkItemLinkPayload) error {
	// Convert payload from app to model representation
	in := app.WorkItemLinkSingle{
		Data: payload.Data,
	}
	modelLink, err := ConvertLinkToModel(in)
	if err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	createdModelLink, err := ctx.Application.WorkItemLinks().Create(ctx.Context, modelLink.SourceID, modelLink.TargetID, modelLink.LinkTypeID, *ctx.CurrentUserIdentityID)
	if err != nil {
		switch reflect.TypeOf(err) {
		case reflect.TypeOf(&goa.ErrorResponse{}):
			return jsonapi.JSONErrorResponse(httpFuncs, goa.ErrBadRequest(err.Error()))
		default:
			return jsonapi.JSONErrorResponse(httpFuncs, err)
		}
	}
	// convert from model to rest representation
	createdAppLink := ConvertLinkFromModel(*createdModelLink)
	if err := enrichLinkSingle(ctx, &createdAppLink); err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	ctx.ResponseData.Header().Set("Location", app.WorkItemLinkHref(createdAppLink.Data.ID))
	return httpFuncs.Created(&createdAppLink)
}

// Create runs the create action.
func (c *WorkItemLinkController) Create(ctx *app.CreateWorkItemLinkContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	linkCtx := newWorkItemLinkContext(ctx.Context, c.db, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, currentUserIdentityID)
	return createWorkItemLink(linkCtx, ctx, ctx.Payload)
}

type deleteWorkItemLinkFuncs interface {
	OK(resp []byte) error
	BadRequest(r *app.JSONAPIErrors) error
	NotFound(r *app.JSONAPIErrors) error
	Unauthorized(r *app.JSONAPIErrors) error
	InternalServerError(r *app.JSONAPIErrors) error
}

func deleteWorkItemLink(ctx *workItemLinkContext, httpFuncs deleteWorkItemLinkFuncs, linkID uuid.UUID) error {
	err := ctx.Application.WorkItemLinks().Delete(ctx.Context, linkID, *ctx.CurrentUserIdentityID)
	if err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	return httpFuncs.OK([]byte{})
}

//
// Delete runs the delete action
func (c *WorkItemLinkController) Delete(ctx *app.DeleteWorkItemLinkContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, currentUserIdentityID)
		return deleteWorkItemLink(linkCtx, ctx, ctx.LinkID)
	})
}

// Show runs the show action.
func (c *WorkItemLinkController) Show(ctx *app.ShowWorkItemLinkContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLink, err := appl.WorkItemLinks().Load(ctx.Context, ctx.LinkID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*modelLink, c.config.GetCacheControlWorkItemLinks, func() error {
			linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, nil)
			// convert to rest representation
			appLink := ConvertLinkFromModel(*modelLink)
			if err := enrichLinkSingle(linkCtx, &appLink); err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			return ctx.OK(&appLink)
		})
	})
}

type updateWorkItemLinkFuncs interface {
	OK(r *app.WorkItemLinkSingle) error
	NotFound(r *app.JSONAPIErrors) error
	BadRequest(r *app.JSONAPIErrors) error
	InternalServerError(r *app.JSONAPIErrors) error
	Unauthorized(r *app.JSONAPIErrors) error
}

func updateWorkItemLink(ctx *workItemLinkContext, httpFuncs updateWorkItemLinkFuncs, payload *app.UpdateWorkItemLinkPayload) error {
	toSave := app.WorkItemLinkSingle{
		Data: payload.Data,
	}
	modelLink, err := ConvertLinkToModel(toSave)
	if err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	savedModelLink, err := ctx.Application.WorkItemLinks().Save(ctx.Context, *modelLink, *ctx.CurrentUserIdentityID)
	if err != nil {
		jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(ctx.Context, err)
		return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
	}
	// Convert the created link type entry into a rest representation
	savedAppLink := ConvertLinkFromModel(*savedModelLink)

	if err := enrichLinkSingle(ctx, &savedAppLink); err != nil {
		return jsonapi.JSONErrorResponse(httpFuncs, err)
	}
	return httpFuncs.OK(&savedAppLink)
}

// Update runs the update action.
func (c *WorkItemLinkController) Update(ctx *app.UpdateWorkItemLinkContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, currentUserIdentityID)
		return updateWorkItemLink(linkCtx, ctx, ctx.Payload)
	})
}

// ConvertLinkFromModel converts a work item from model to REST representation
func ConvertLinkFromModel(t link.WorkItemLink) app.WorkItemLinkSingle {
	var converted = app.WorkItemLinkSingle{
		Data: &app.WorkItemLinkData{
			Type: link.EndpointWorkItemLinks,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkAttributes{
				CreatedAt: &t.CreatedAt,
				UpdatedAt: &t.UpdatedAt,
				Version:   &t.Version,
			},
			Relationships: &app.WorkItemLinkRelationships{
				LinkType: &app.RelationWorkItemLinkType{
					Data: &app.RelationWorkItemLinkTypeData{
						Type: link.EndpointWorkItemLinkTypes,
						ID:   t.LinkTypeID,
					},
				},
				Source: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: link.EndpointWorkItems,
						ID:   t.SourceID,
					},
				},
				Target: &app.RelationWorkItem{
					Data: &app.RelationWorkItemData{
						Type: link.EndpointWorkItems,
						ID:   t.TargetID,
					},
				},
			},
		},
	}
	return converted
}

// ConvertLinkToModel converts the incoming app representation of a work item link to the model layout.
// Values are only overwrriten if they are set in "in", otherwise the values in "out" remain.
// NOTE: Only the LinkTypeID, SourceID, and TargetID fields will be set.
//       You need to preload the elements after calling this function.
func ConvertLinkToModel(appLink app.WorkItemLinkSingle) (*link.WorkItemLink, error) {
	modelLink := link.WorkItemLink{}
	attrs := appLink.Data.Attributes
	rel := appLink.Data.Relationships
	if appLink.Data.ID != nil {
		modelLink.ID = *appLink.Data.ID
	}

	if attrs != nil && attrs.Version != nil {
		modelLink.Version = *attrs.Version
	}

	if rel != nil && rel.LinkType != nil && rel.LinkType.Data != nil {
		modelLink.LinkTypeID = rel.LinkType.Data.ID
	}

	if rel != nil && rel.Source != nil && rel.Source.Data != nil {
		d := rel.Source.Data
		// The the work item id MUST NOT be empty
		if d.ID == uuid.Nil {
			return nil, errors.NewBadParameterError("data.relationships.source.data.id", d.ID)
		}
		modelLink.SourceID = d.ID
	}

	if rel != nil && rel.Target != nil && rel.Target.Data != nil {
		d := rel.Target.Data
		// If the the target type is not nil, it MUST be "workitems"
		// The the work item id MUST NOT be empty
		if d.ID == uuid.Nil {
			return nil, errors.NewBadParameterError("data.relationships.target.data.id", d.ID)
		}
		modelLink.TargetID = d.ID
	}

	return &modelLink, nil
}
