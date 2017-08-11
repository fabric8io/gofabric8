package controller

import (
	"fmt"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// WorkItemRelationshipsLinksController implements the work-item-relationships-links resource.
type WorkItemRelationshipsLinksController struct {
	*goa.Controller
	db     application.DB
	config WorkItemRelationshipsLinksControllerConfig
}

// WorkItemRelationshipsLinksControllerConfig the config interface for the WorkItemRelationshipsLinksController
type WorkItemRelationshipsLinksControllerConfig interface {
	GetCacheControlWorkItemLinks() string
}

// NewWorkItemRelationshipsLinksController creates a work-item-relationships-links controller.
func NewWorkItemRelationshipsLinksController(service *goa.Service, db application.DB, config WorkItemRelationshipsLinksControllerConfig) *WorkItemRelationshipsLinksController {
	return &WorkItemRelationshipsLinksController{
		Controller: service.NewController("WorkItemRelationshipsLinksController"),
		db:         db,
		config:     config,
	}
}

func parseWorkItemIDToUint64(wiIDStr string) (uint64, error) {
	wiID, err := strconv.ParseUint(wiIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid work item ID \"%s\": %s", wiIDStr, err.Error())
	}
	return wiID, nil
}

// Create runs the create action.
func (c *WorkItemRelationshipsLinksController) Create(ctx *app.CreateWorkItemRelationshipsLinksContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		// Check that current work item does indeed exist
		wi, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(ctx, err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// Check that the source ID of the link is the same as the current work
		// item ID.
		src, _ := getSrcTgt(ctx.Payload.Data)
		if src != nil && *src != wi.ID {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest(fmt.Sprintf("data.relationships.source.data.id is \"%s\" but must be \"%s\"", ctx.Payload.Data.Relationships.Source.Data.ID.String(), wi.ID.String())))
			return ctx.BadRequest(jerrors)
		}
		// If no source is specified we pre-fill the source field of the payload
		// with the current work item ID from the URL. This is for convenience.
		if src == nil {
			if ctx.Payload.Data.Relationships == nil {
				ctx.Payload.Data.Relationships = &app.WorkItemLinkRelationships{}
			}
			if ctx.Payload.Data.Relationships.Source == nil {
				ctx.Payload.Data.Relationships.Source = &app.RelationWorkItem{}
			}
			if ctx.Payload.Data.Relationships.Source.Data == nil {
				ctx.Payload.Data.Relationships.Source.Data = &app.RelationWorkItemData{}
			}
			ctx.Payload.Data.Relationships.Source.Data.ID = wi.ID
			ctx.Payload.Data.Relationships.Source.Data.Type = link.EndpointWorkItems
		}
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, currentUserIdentityID)
		return createWorkItemLink(linkCtx, ctx, ctx.Payload)
	})
}

// List runs the list action.
func (c *WorkItemRelationshipsLinksController) List(ctx *app.ListWorkItemRelationshipsLinksContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelLinks, err := appl.WorkItemLinks().ListByWorkItem(ctx.Context, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntities(modelLinks, c.config.GetCacheControlWorkItemLinks, func() error {
			appLinks := app.WorkItemLinkList{}
			appLinks.Data = make([]*app.WorkItemLinkData, len(modelLinks))
			for index, modelLink := range modelLinks {
				appLink := ConvertLinkFromModel(modelLink)
				appLinks.Data[index] = appLink.Data
			}
			// TODO: When adding pagination, this must not be len(rows) but
			// the overall total number of elements from all pages.
			appLinks.Meta = &app.WorkItemLinkListMeta{
				TotalCount: len(modelLinks),
			}
			linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkHref, nil)
			if err := enrichLinkList(linkCtx, &appLinks); err != nil {
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			return ctx.OK(&appLinks)
		})
	})
}

func getSrcTgt(wilData *app.WorkItemLinkData) (*uuid.UUID, *uuid.UUID) {
	var src, tgt *uuid.UUID
	if wilData != nil && wilData.Relationships != nil {
		if wilData.Relationships.Source != nil && wilData.Relationships.Source.Data != nil {
			src = &wilData.Relationships.Source.Data.ID
		}
		if wilData.Relationships.Target != nil && wilData.Relationships.Target.Data != nil {
			tgt = &wilData.Relationships.Target.Data.ID
		}
	}
	return src, tgt
}
