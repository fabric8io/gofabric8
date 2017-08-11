package controller

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Defines the constants to be used in json api "type" attribute
const (
	APIStringTypeUser         = "identities"
	APIStringTypeWorkItem     = "workitems"
	APIStringTypeWorkItemType = "workitemtypes"
	none                      = "none"
)

// WorkitemController implements the workitem resource.
type WorkitemController struct {
	*goa.Controller
	db           application.DB
	config       WorkItemControllerConfig
	notification notification.Channel
}

// WorkItemControllerConfig the config interface for the WorkitemController
type WorkItemControllerConfig interface {
	GetCacheControlWorkItems() string
}

// NewWorkitemController creates a workitem controller.
func NewWorkitemController(service *goa.Service, db application.DB, config WorkItemControllerConfig) *WorkitemController {
	return NewNotifyingWorkitemController(service, db, &notification.DevNullChannel{}, config)
}

// NewNotifyingWorkitemController creates a workitem controller with notification broadcast.
func NewNotifyingWorkitemController(service *goa.Service, db application.DB, notificationChannel notification.Channel, config WorkItemControllerConfig) *WorkitemController {
	n := notificationChannel
	if n == nil {
		n = &notification.DevNullChannel{}
	}
	return &WorkitemController{
		Controller:   service.NewController("WorkitemController"),
		db:           db,
		notification: n,
		config:       config}
}

// Returns true if the user is the work item creator or space collaborator
func authorizeWorkitemEditor(ctx context.Context, db application.DB, spaceID uuid.UUID, creatorID string, editorID string) (bool, error) {
	if editorID == creatorID {
		return true, nil
	}
	authorized, err := authz.Authorize(ctx, spaceID.String())
	if err != nil {
		return false, errors.NewUnauthorizedError(err.Error())
	}
	return authorized, nil
}

// Update does PATCH workitem
func (c *WorkitemController) Update(ctx *app.UpdateWorkitemContext) error {
	if ctx.Payload == nil || ctx.Payload.Data == nil || ctx.Payload.Data.ID == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("missing data.ID element in request", nil))
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}

	var wi *workitem.WorkItem
	err = application.Transactional(c.db, func(appl application.Application) error {
		wi, err = appl.WorkItems().LoadByID(ctx, *ctx.Payload.Data.ID)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("Failed to load work item with id %v", *ctx.Payload.Data.ID))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	creator := wi.Fields[workitem.SystemCreator]
	if creator == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.New("work item doesn't have creator")))
	}
	authorized, err := authorizeWorkitemEditor(ctx, c.db, wi.SpaceID, creator.(string), currentUserIdentityID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to access the space"))
	}
	result := application.Transactional(c.db, func(appl application.Application) error {
		// Type changes of WI are not allowed which is why we overwrite it the
		// type with the old one after the WI has been converted.
		oldType := wi.Type
		err = ConvertJSONAPIToWorkItem(ctx, ctx.Method, appl, *ctx.Payload.Data, wi, wi.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		wi.Type = oldType
		wi, err = appl.WorkItems().Save(ctx, wi.SpaceID, *wi, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error updating work item"))
		}
		hasChildren := workItemIncludeHasChildren(appl, ctx)
		wi2 := ConvertWorkItem(ctx.RequestData, *wi, hasChildren)
		resp := &app.WorkItemSingle{
			Data: wi2,
			Links: &app.WorkItemLinks{
				Self: buildAbsoluteURL(ctx.RequestData),
			},
		}

		ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
		return ctx.OK(resp)
	})
	if ctx.ResponseData.Status == 200 {
		c.notification.Send(ctx, notification.NewWorkItemUpdated(ctx.Payload.Data.ID.String()))
	}
	return result
}

// Show does GET workitem
func (c *WorkitemController) Show(ctx *app.ShowWorkitemContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		wi, err := appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, fmt.Sprintf("Fail to load work item with id %v", ctx.WiID)))
		}
		return ctx.ConditionalRequest(*wi, c.config.GetCacheControlWorkItems, func() error {
			comments := workItemIncludeCommentsAndTotal(ctx, c.db, ctx.WiID)
			hasChildren := workItemIncludeHasChildren(appl, ctx)
			wi2 := ConvertWorkItem(ctx.RequestData, *wi, comments, hasChildren)
			resp := &app.WorkItemSingle{
				Data: wi2,
			}
			return ctx.OK(resp)

		})
	})
}

// Delete does DELETE workitem
func (c *WorkitemController) Delete(ctx *app.DeleteWorkitemContext) error {

	// Temporarly disabled, See https://github.com/fabric8-services/fabric8-wit/issues/1036
	if true {
		return ctx.MethodNotAllowed()
	}
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	var wi *workitem.WorkItem
	err = application.Transactional(c.db, func(appl application.Application) error {
		wi, err = appl.WorkItems().LoadByID(ctx, ctx.WiID)
		if err != nil {
			return errs.Wrap(err, fmt.Sprintf("Failed to load work item with id %v", ctx.WiID))
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not authorized to access the space"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItems().Delete(ctx, ctx.WiID, *currentUserIdentityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "error deleting work item %s", ctx.WiID))
		}
		if err := appl.WorkItemLinks().DeleteRelatedLinks(ctx, ctx.WiID, *currentUserIdentityID); err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "failed to delete work item links related to work item %s", ctx.WiID))
		}
		return ctx.OK([]byte{})
	})
}

// Time is default value if no UpdatedAt field is found
func updatedAt(wi workitem.WorkItem) time.Time {
	var t time.Time
	if ua, ok := wi.Fields[workitem.SystemUpdatedAt]; ok {
		t = ua.(time.Time)
	}
	return t.Truncate(time.Second)
}

func lastModified(wi workitem.WorkItem) string {
	return lastModifiedTime(updatedAt(wi))
}

func lastModifiedTime(t time.Time) string {
	return t.Format(time.RFC1123)
}

func findLastModified(wis []workitem.WorkItem) time.Time {
	var t time.Time
	for _, wi := range wis {
		lm := updatedAt(wi)
		if lm.After(t) {
			t = lm
		}
	}
	return t
}

// ConvertJSONAPIToWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertJSONAPIToWorkItem(ctx context.Context, method string, appl application.Application, source app.WorkItem, target *workitem.WorkItem, spaceID uuid.UUID) error {
	// construct default values from input WI
	version, err := getVersion(source.Attributes["version"])
	if err != nil {
		return err
	}
	target.Version = version

	if source.Relationships != nil && source.Relationships.Assignees != nil {
		if source.Relationships.Assignees.Data == nil {
			delete(target.Fields, workitem.SystemAssignees)
		} else {
			var ids []string
			for _, d := range source.Relationships.Assignees.Data {
				assigneeUUID, err := uuid.FromString(*d.ID)
				if err != nil {
					return errors.NewBadParameterError("data.relationships.assignees.data.id", *d.ID)
				}
				if ok := appl.Identities().IsValid(ctx, assigneeUUID); !ok {
					return errors.NewBadParameterError("data.relationships.assignees.data.id", *d.ID)
				}
				ids = append(ids, assigneeUUID.String())
			}
			target.Fields[workitem.SystemAssignees] = ids
		}
	}
	if source.Relationships != nil {
		if source.Relationships.Iteration == nil || (source.Relationships.Iteration != nil && source.Relationships.Iteration.Data == nil) {
			log.Debug(ctx, map[string]interface{}{
				"wi_id":    target.ID,
				"space_id": spaceID,
			}, "assigning the work item to the root iteration of the space.")
			rootIteration, err := appl.Iterations().Root(ctx, spaceID)
			if err != nil {
				return errors.NewBadParameterError("space", spaceID).Expected("valid space ID")
			}
			if method == http.MethodPost {
				target.Fields[workitem.SystemIteration] = rootIteration.ID.String()
			} else if method == http.MethodPatch {
				if source.Relationships.Iteration != nil && source.Relationships.Iteration.Data == nil {
					target.Fields[workitem.SystemIteration] = rootIteration.ID.String()
				}
			}
		} else if source.Relationships.Iteration != nil && source.Relationships.Iteration.Data != nil {
			d := source.Relationships.Iteration.Data
			iterationUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.iteration.data.id", *d.ID)
			}
			if err := appl.Iterations().CheckExists(ctx, iterationUUID.String()); err != nil {
				return errors.NewNotFoundError("data.relationships.iteration.data.id", *d.ID)
			}
			target.Fields[workitem.SystemIteration] = iterationUUID.String()
		}
	}

	if source.Relationships != nil {
		if source.Relationships.Area == nil || (source.Relationships.Area != nil && source.Relationships.Area.Data == nil) {
			log.Debug(ctx, map[string]interface{}{
				"wi_id":    target.ID,
				"space_id": spaceID,
			}, "assigning the work item to the root area of the space.")
			rootArea, err := appl.Areas().Root(ctx, spaceID)
			if err != nil {
				return errors.NewBadParameterError("space", spaceID).Expected("valid space ID")
			}
			if method == http.MethodPost {
				target.Fields[workitem.SystemArea] = rootArea.ID.String()
			} else if method == http.MethodPatch {
				if source.Relationships.Area != nil && source.Relationships.Area.Data == nil {
					target.Fields[workitem.SystemArea] = rootArea.ID.String()
				}
			}
		} else if source.Relationships.Area != nil && source.Relationships.Area.Data != nil {
			d := source.Relationships.Area.Data
			areaUUID, err := uuid.FromString(*d.ID)
			if err != nil {
				return errors.NewBadParameterError("data.relationships.area.data.id", *d.ID)
			}
			if err := appl.Areas().CheckExists(ctx, areaUUID.String()); err != nil {
				cause := errs.Cause(err)
				switch cause.(type) {
				case errors.NotFoundError:
					return errors.NewNotFoundError("data.relationships.area.data.id", *d.ID)
				default:
					return errs.Wrapf(err, "unknown error when verifying the area id %s", *d.ID)
				}
			}
			target.Fields[workitem.SystemArea] = areaUUID.String()
		}
	}

	if source.Relationships != nil && source.Relationships.BaseType != nil {
		if source.Relationships.BaseType.Data != nil {
			target.Type = source.Relationships.BaseType.Data.ID
		}
	}

	for key, val := range source.Attributes {
		// convert legacy description to markup content
		if key == workitem.SystemDescription {
			if m := rendering.NewMarkupContentFromValue(val); m != nil {
				// if no description existed before, set the new one
				if target.Fields[key] == nil {
					target.Fields[key] = *m
				} else {
					// only update the 'description' field in the existing description
					existingDescription := target.Fields[key].(rendering.MarkupContent)
					existingDescription.Content = (*m).Content
					target.Fields[key] = existingDescription
				}
			}
		} else if key == workitem.SystemDescriptionMarkup {
			markup := val.(string)
			// if no description existed before, set the markup in a new one
			if target.Fields[workitem.SystemDescription] == nil {
				target.Fields[workitem.SystemDescription] = rendering.MarkupContent{Markup: markup}
			} else {
				// only update the 'description' field in the existing description
				existingDescription := target.Fields[workitem.SystemDescription].(rendering.MarkupContent)
				existingDescription.Markup = markup
				target.Fields[workitem.SystemDescription] = existingDescription
			}
		} else if key == workitem.SystemCodebase {
			if m, err := codebase.NewCodebaseContentFromValue(val); err == nil {
				setupCodebase(appl, m, spaceID)
				target.Fields[key] = *m
			} else {
				return err
			}
		} else {
			target.Fields[key] = val
		}
	}
	if description, ok := target.Fields[workitem.SystemDescription].(rendering.MarkupContent); ok {
		// verify the description markup
		if !rendering.IsMarkupSupported(description.Markup) {
			return errors.NewBadParameterError("data.relationships.attributes[system.description].markup", description.Markup)
		}
	}
	return nil
}

// setupCodebase is the link between CodebaseContent & Codebase
// setupCodebase creates a codebase and saves it's ID in CodebaseContent
// for future use
func setupCodebase(appl application.Application, cb *codebase.Content, spaceID uuid.UUID) error {
	if cb.CodebaseID == "" {
		defaultStackID := "java-centos"
		newCodeBase := codebase.Codebase{
			SpaceID: spaceID,
			Type:    "git",
			URL:     cb.Repository,
			StackID: &defaultStackID,
			//TODO: Think of making stackID dynamic value (from analyzer)
		}
		existingCB, err := appl.Codebases().LoadByRepo(context.Background(), spaceID, cb.Repository)
		if existingCB != nil {
			cb.CodebaseID = existingCB.ID.String()
			return nil
		}
		err = appl.Codebases().Create(context.Background(), &newCodeBase)
		if err != nil {
			return errors.NewInternalError(context.Background(), err)
		}
		cb.CodebaseID = newCodeBase.ID.String()
	}
	return nil
}

func getVersion(version interface{}) (int, error) {
	if version != nil {
		v, err := strconv.Atoi(fmt.Sprintf("%v", version))
		if err != nil {
			return -1, errors.NewBadParameterError("data.attributes.version", version)
		}
		return v, nil
	}
	return -1, nil
}

// WorkItemConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
type WorkItemConvertFunc func(*goa.RequestData, *workitem.WorkItem, *app.WorkItem)

// ConvertWorkItems is responsible for converting given []WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItems(request *goa.RequestData, wis []workitem.WorkItem, additional ...WorkItemConvertFunc) []*app.WorkItem {
	ops := []*app.WorkItem{}
	for _, wi := range wis {
		ops = append(ops, ConvertWorkItem(request, wi, additional...))
	}
	return ops
}

// ConvertWorkItem is responsible for converting given WorkItem model object into a
// response resource object by jsonapi.org specifications
func ConvertWorkItem(request *goa.RequestData, wi workitem.WorkItem, additional ...WorkItemConvertFunc) *app.WorkItem {
	// construct default values from input WI
	relatedURL := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID))
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(wi.SpaceID.String()))
	witRelatedURL := rest.AbsoluteURL(request, app.WorkitemtypeHref(wi.SpaceID.String(), wi.Type))

	op := &app.WorkItem{
		ID:   &wi.ID,
		Type: APIStringTypeWorkItem,
		Attributes: map[string]interface{}{
			workitem.SystemVersion: wi.Version,
			workitem.SystemNumber:  wi.Number,
		},
		Relationships: &app.WorkItemRelationships{
			BaseType: &app.RelationBaseType{
				Data: &app.BaseTypeData{
					ID:   wi.Type,
					Type: APIStringTypeWorkItemType,
				},
				Links: &app.GenericLinks{
					Self: &witRelatedURL,
				},
			},
			Space: app.NewSpaceRelation(wi.SpaceID, spaceRelatedURL),
		},
		Links: &app.GenericLinksForWorkItem{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}

	// Move fields into Relationships or Attributes as needed
	// TODO: Loop based on WorkItemType and match against Field.Type instead of directly to field value
	for name, val := range wi.Fields {
		switch name {
		case workitem.SystemAssignees:
			if val != nil {
				userID := val.([]interface{})
				op.Relationships.Assignees = &app.RelationGenericList{
					Data: ConvertUsersSimple(request, userID),
				}
			}
		case workitem.SystemCreator:
			if val != nil {
				userID := val.(string)
				op.Relationships.Creator = &app.RelationGeneric{
					Data: ConvertUserSimple(request, userID),
				}
			}
		case workitem.SystemIteration:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Iteration = &app.RelationGeneric{
					Data: ConvertIterationSimple(request, valStr),
				}
			}
		case workitem.SystemArea:
			if val != nil {
				valStr := val.(string)
				op.Relationships.Area = &app.RelationGeneric{
					Data: ConvertAreaSimple(request, valStr),
				}
			}

		case workitem.SystemTitle:
			// 'HTML escape' the title to prevent script injection
			op.Attributes[name] = html.EscapeString(val.(string))
		case workitem.SystemDescription:
			description := rendering.NewMarkupContentFromValue(val)
			if description != nil {
				op.Attributes[name] = (*description).Content
				op.Attributes[workitem.SystemDescriptionMarkup] = (*description).Markup
				// let's include the rendered description while 'HTML escaping' it to prevent script injection
				op.Attributes[workitem.SystemDescriptionRendered] =
					rendering.RenderMarkupToHTML(html.EscapeString((*description).Content), (*description).Markup)
			}
		case workitem.SystemCodebase:
			if val != nil {
				op.Attributes[name] = val
				// TODO: Following format is TBD and hence commented out
				cb := val.(codebase.Content)
				editURL := rest.AbsoluteURL(request, app.CodebaseHref(cb.CodebaseID)+"/edit")
				op.Links.EditCodebase = &editURL
			}
		default:
			op.Attributes[name] = val
		}
	}
	if op.Relationships.Assignees == nil {
		op.Relationships.Assignees = &app.RelationGenericList{Data: nil}
	}
	if op.Relationships.Iteration == nil {
		op.Relationships.Iteration = &app.RelationGeneric{Data: nil}
	}
	if op.Relationships.Area == nil {
		op.Relationships.Area = &app.RelationGeneric{Data: nil}
	}
	// Always include Comments Link, but optionally use workItemIncludeCommentsAndTotal
	workItemIncludeComments(request, &wi, op)
	workItemIncludeChildren(request, &wi, op)
	for _, add := range additional {
		add(request, &wi, op)
	}
	return op
}

// workItemIncludeHasChildren adds meta information about existing children
func workItemIncludeHasChildren(appl application.Application, ctx context.Context) WorkItemConvertFunc {
	// TODO: Wrap ctx in a Timeout context?
	return func(request *goa.RequestData, wi *workitem.WorkItem, wi2 *app.WorkItem) {
		var hasChildren bool
		var err error
		repo := appl.WorkItemLinks()
		if repo != nil {
			hasChildren, err = appl.WorkItemLinks().WorkItemHasChildren(ctx, wi.ID)
			log.Info(ctx, map[string]interface{}{"wi_id": wi.ID}, "Work item has children: %t", hasChildren)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"wi_id": wi.ID,
					"err":   err,
				}, "unable to find out if work item has children: %s", wi.ID)
				// enforce to have no children
				hasChildren = false
			}
		}
		if wi2.Relationships.Children == nil {
			wi2.Relationships.Children = &app.RelationGeneric{}
		}
		wi2.Relationships.Children.Meta = map[string]interface{}{
			"hasChildren": hasChildren,
		}

	}
}

// ListChildren runs the list action.
func (c *WorkitemController) ListChildren(ctx *app.ListChildrenWorkitemContext) error {
	// WorkItemChildrenController_List: start_implement

	var additionalQuery []string
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(appl application.Application) error {
		result, tc, err := appl.WorkItemLinks().ListWorkItemChildren(ctx, ctx.WiID, &offset, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "unable to list work item children"))
		}
		count := int(tc)
		return ctx.ConditionalEntities(result, c.config.GetCacheControlWorkItems, func() error {
			hasChildren := workItemIncludeHasChildren(appl, ctx)
			response := app.WorkItemList{
				Links: &app.PagingLinks{},
				Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
				Data:  ConvertWorkItems(ctx.RequestData, result, hasChildren),
			}
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, additionalQuery...)
			return ctx.OK(&response)
		})
	})
}

// workItemIncludeChildren adds relationship about children to workitem (include totalCount)
func workItemIncludeChildren(request *goa.RequestData, wi *workitem.WorkItem, wi2 *app.WorkItem) {
	childrenRelated := rest.AbsoluteURL(request, app.WorkitemHref(wi.ID)) + "/children"
	if wi2.Relationships.Children == nil {
		wi2.Relationships.Children = &app.RelationGeneric{}
	}
	wi2.Relationships.Children.Links = &app.GenericLinks{
		Related: &childrenRelated,
	}
}
