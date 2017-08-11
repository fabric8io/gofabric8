package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/satori/go.uuid"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

const (
	sourceLinkTypesRouteEnd = "/source-link-types"
	targetLinkTypesRouteEnd = "/target-link-types"
)

// WorkitemtypeController implements the workitemtype resource.
type WorkitemtypeController struct {
	*goa.Controller
	db     application.DB
	config WorkItemControllerConfiguration
}

type WorkItemControllerConfiguration interface {
	GetCacheControlWorkItemTypes() string
}

// NewWorkitemtypeController creates a workitemtype controller.
func NewWorkitemtypeController(service *goa.Service, db application.DB, config WorkItemControllerConfiguration) *WorkitemtypeController {
	return &WorkitemtypeController{
		Controller: service.NewController("WorkitemtypeController"),
		db:         db,
		config:     config,
	}
}

// Show runs the show action.
func (c *WorkitemtypeController) Show(ctx *app.ShowWorkitemtypeContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		witModel, err := appl.WorkItemTypes().Load(ctx.Context, ctx.SpaceID, ctx.WitID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*witModel, c.config.GetCacheControlWorkItemTypes, func() error {
			witData := ConvertWorkItemTypeFromModel(ctx.RequestData, witModel)
			wit := &app.WorkItemTypeSingle{Data: &witData}
			return ctx.OK(wit)
		})
	})
}

// Create runs the create action.
func (c *WorkitemtypeController) Create(ctx *app.CreateWorkitemtypeContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		space, err := appl.Spaces().Load(ctx, ctx.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		if !uuid.Equal(*currentUserIdentityID, space.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     ctx.SpaceID,
				"space_owner":  space.OwnerId,
				"current_user": *currentUserIdentityID,
			}, "user is not the space owner")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}
		var fields = map[string]app.FieldDefinition{}
		for key, fd := range ctx.Payload.Data.Attributes.Fields {
			fields[key] = *fd
		}
		// Set the space to the Payload
		if ctx.Payload.Data != nil && ctx.Payload.Data.Relationships != nil {
			// We overwrite or use the space ID in the URL to set the space of this WI
			spaceSelfURL := rest.AbsoluteURL(ctx.RequestData, app.SpaceHref(ctx.SpaceID.String()))
			ctx.Payload.Data.Relationships.Space = app.NewSpaceRelation(ctx.SpaceID, spaceSelfURL)
		}
		modelFields, err := ConvertFieldDefinitionsToModel(fields)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		witTypeModel, err := appl.WorkItemTypes().Create(
			ctx.Context,
			*ctx.Payload.Data.Relationships.Space.Data.ID,
			ctx.Payload.Data.ID,
			ctx.Payload.Data.Attributes.ExtendedTypeName,
			ctx.Payload.Data.Attributes.Name,
			ctx.Payload.Data.Attributes.Description,
			ctx.Payload.Data.Attributes.Icon,
			modelFields)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		witData := ConvertWorkItemTypeFromModel(ctx.RequestData, witTypeModel)
		wit := &app.WorkItemTypeSingle{Data: &witData}
		ctx.ResponseData.Header().Set("Location", app.WorkitemtypeHref(*ctx.Payload.Data.Relationships.Space.Data.ID, wit.Data.ID))
		return ctx.Created(wit)
	})
}

// List runs the list action
func (c *WorkitemtypeController) List(ctx *app.ListWorkitemtypeContext) error {
	log.Debug(ctx, map[string]interface{}{"space_id": ctx.SpaceID}, "Listing work item types per space")
	start, limit, err := parseLimit(ctx.Page)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Could not parse paging"))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		witModelsOrig, err := appl.WorkItemTypes().List(ctx.Context, ctx.SpaceID, start, &limit)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work item types"))
		}
		// Remove "planneritem" from the list of WITs
		witModels := []workitem.WorkItemType{}
		for _, wit := range witModelsOrig {
			if wit.ID != workitem.SystemPlannerItem {
				witModels = append(witModels, wit)
			}
		}
		return ctx.ConditionalEntities(witModels, c.config.GetCacheControlWorkItemTypes, func() error {
			// TEMP!!!!! Until Space Template can setup a Space, redirect to SystemSpace WITs if non are found
			// for the space.
			if len(witModels) == 0 {
				witModels, err = appl.WorkItemTypes().List(ctx.Context, space.SystemSpace, start, &limit)
				if err != nil {
					return jsonapi.JSONErrorResponse(ctx, errs.Wrap(err, "Error listing work item types"))
				}
			}
			// convert from model to app
			result := &app.WorkItemTypeList{}
			result.Data = make([]*app.WorkItemTypeData, len(witModels))
			for index, value := range witModels {
				wit := ConvertWorkItemTypeFromModel(ctx.RequestData, &value)
				result.Data[index] = &wit
			}
			return ctx.OK(result)
		})
	})
}

// converts from models to app representation
func ConvertWorkItemTypeFromModel(request *goa.RequestData, t *workitem.WorkItemType) app.WorkItemTypeData {
	spaceSelfURL := rest.AbsoluteURL(request, app.SpaceHref(t.SpaceID.String()))
	id := t.ID
	createdAt := t.CreatedAt.UTC()
	updatedAt := t.UpdatedAt.UTC()
	var converted = app.WorkItemTypeData{
		Type: "workitemtypes",
		ID:   &id,
		Attributes: &app.WorkItemTypeAttributes{
			CreatedAt:   &createdAt,
			UpdatedAt:   &updatedAt,
			Version:     &t.Version,
			Description: t.Description,
			Icon:        t.Icon,
			Name:        t.Name,
			Fields:      map[string]*app.FieldDefinition{},
		},
		Relationships: &app.WorkItemTypeRelationships{
			Space: app.NewSpaceRelation(t.SpaceID, spaceSelfURL),
		},
	}
	for name, def := range t.Fields {
		ct := convertFieldTypeFromModel(def.Type)
		converted.Attributes.Fields[name] = &app.FieldDefinition{
			Required:    def.Required,
			Label:       def.Label,
			Description: def.Description,
			Type:        &ct,
		}
	}
	return converted
}

// converts the field type from modesl to app representation
func convertFieldTypeFromModel(t workitem.FieldType) app.FieldType {
	result := app.FieldType{}
	result.Kind = string(t.GetKind())
	switch t2 := t.(type) {
	case workitem.ListType:
		kind := string(t2.ComponentType.GetKind())
		result.ComponentType = &kind
	case workitem.EnumType:
		kind := string(t2.BaseType.GetKind())
		result.BaseType = &kind
		result.Values = t2.Values
	}

	return result
}

func convertFieldTypeToModel(t app.FieldType) (workitem.FieldType, error) {
	kind, err := workitem.ConvertStringToKind(t.Kind)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	switch *kind {
	case workitem.KindList:
		componentType, err := workitem.ConvertAnyToKind(*t.ComponentType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !componentType.IsSimpleType() {
			return nil, fmt.Errorf("Component type is not list type: %T", componentType)
		}
		return workitem.ListType{workitem.SimpleType{*kind}, workitem.SimpleType{*componentType}}, nil
	case workitem.KindEnum:
		bt, err := workitem.ConvertAnyToKind(*t.BaseType)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		if !bt.IsSimpleType() {
			return nil, fmt.Errorf("baseType type is not list type: %T", bt)
		}
		baseType := workitem.SimpleType{*bt}

		values := t.Values
		converted, err := workitem.ConvertList(func(ft workitem.FieldType, element interface{}) (interface{}, error) {
			return ft.ConvertToModel(element)
		}, baseType, values)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		return workitem.EnumType{workitem.SimpleType{*kind}, baseType, converted}, nil
	default:
		return workitem.SimpleType{*kind}, nil
	}
}

func ConvertFieldDefinitionsToModel(fields map[string]app.FieldDefinition) (map[string]workitem.FieldDefinition, error) {
	modelFields := map[string]workitem.FieldDefinition{}
	// now process new fields, checking whether they are ok to add.
	for field, definition := range fields {
		ct, err := convertFieldTypeToModel(*definition.Type)
		if err != nil {
			return nil, errs.WithStack(err)
		}
		converted := workitem.FieldDefinition{
			Label:       definition.Label,
			Description: definition.Description,
			Required:    definition.Required,
			Type:        ct,
		}
		modelFields[field] = converted
	}
	return modelFields, nil
}
