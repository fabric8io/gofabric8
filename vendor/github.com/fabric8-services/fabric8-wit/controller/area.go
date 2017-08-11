package controller

import (
	"fmt"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// AreaController implements the area resource.
type AreaController struct {
	*goa.Controller
	db     application.DB
	config AreaControllerConfiguration
}

// AreaControllerConfiguration the configuration for the AreaController
type AreaControllerConfiguration interface {
	GetCacheControlAreas() string
}

// NewAreaController creates a area controller.
func NewAreaController(service *goa.Service, db application.DB, config AreaControllerConfiguration) *AreaController {
	return &AreaController{
		Controller: service.NewController("AreaController"),
		db:         db,
		config:     config}
}

// ShowChildren runs the show-children action
func (c *AreaController) ShowChildren(ctx *app.ShowChildrenAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		parentArea, err := appl.Areas().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		children, err := appl.Areas().ListChildren(ctx, parentArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalEntities(children, c.config.GetCacheControlAreas, func() error {
			res := &app.AreaList{}
			res.Data = ConvertAreas(appl, ctx.RequestData, children, addResolvedPath)
			return ctx.OK(res)
		})
	})
}

// CreateChild runs the create-child action.
func (c *AreaController) CreateChild(ctx *app.CreateChildAreaContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parentID, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		parent, err := appl.Areas().Load(ctx, parentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		s, err := appl.Spaces().Load(ctx, parent.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
		}
		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     s.ID,
				"space_owner":  s.OwnerId,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}

		reqArea := ctx.Payload.Data
		if reqArea.Attributes.Name == nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil"))
		}

		childPath := append(parent.Path, parent.ID)
		newArea := area.Area{
			SpaceID: parent.SpaceID,
			Path:    childPath,
			Name:    *reqArea.Attributes.Name,
		}

		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.AreaSingle{
			Data: ConvertArea(appl, ctx.RequestData, newArea, addResolvedPath),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.AreaHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// Show runs the show action.
func (c *AreaController) Show(ctx *app.ShowAreaContext) error {
	id, err := uuid.FromString(ctx.ID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		a, err := appl.Areas().Load(ctx, id)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*a, c.config.GetCacheControlAreas, func() error {
			res := &app.AreaSingle{}
			res.Data = ConvertArea(appl, ctx.RequestData, *a, addResolvedPath)
			return ctx.OK(res)
		})
	})
}

// addResolvedPath resolves the path in the form of /area1/area2/area3
func addResolvedPath(appl application.Application, req *goa.RequestData, mArea *area.Area, sArea *app.Area) error {
	pathResolved, error := getResolvePath(appl, mArea)
	sArea.Attributes.ParentPathResolved = pathResolved
	return error
}

func getResolvePath(appl application.Application, a *area.Area) (*string, error) {
	parentUuids := a.Path
	parentAreas, err := appl.Areas().LoadMultiple(context.Background(), parentUuids)
	if err != nil {
		return nil, err
	}
	pathResolved := ""
	for _, a := range parentUuids {
		area := getAreaByID(a, parentAreas)
		if area == nil {
			continue
		}
		pathResolved = pathResolved + path.SepInService + area.Name
	}

	// Add the leading "/" in the "area1/area2/area3" styled path
	if pathResolved == "" {
		pathResolved = "/"
	}
	return &pathResolved, nil
}

func getAreaByID(id uuid.UUID, areas []area.Area) *area.Area {
	for _, a := range areas {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

// AreaConvertFunc is a open ended function to add additional links/data/relations to a area during
// convertion from internal to API
type AreaConvertFunc func(application.Application, *goa.RequestData, *area.Area, *app.Area) error

// ConvertAreas converts between internal and external REST representation
func ConvertAreas(appl application.Application, request *goa.RequestData, areas []area.Area, additional ...AreaConvertFunc) []*app.Area {
	var is = []*app.Area{}
	for _, i := range areas {
		is = append(is, ConvertArea(appl, request, i, additional...))
	}
	return is
}

// ConvertArea converts between internal and external REST representation
func ConvertArea(appl application.Application, request *goa.RequestData, ar area.Area, additional ...AreaConvertFunc) *app.Area {
	areaType := area.APIStringTypeAreas
	spaceID := ar.SpaceID.String()
	relatedURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID))
	childURL := rest.AbsoluteURL(request, app.AreaHref(ar.ID)+"/children")
	spaceRelatedURL := rest.AbsoluteURL(request, app.SpaceHref(spaceID))
	pathToTopMostParent := ar.Path.String() // /uuid1/uuid2/uuid3s
	i := &app.Area{
		Type: areaType,
		ID:   &ar.ID,
		Attributes: &app.AreaAttributes{
			Name:       &ar.Name,
			CreatedAt:  &ar.CreatedAt,
			UpdatedAt:  &ar.UpdatedAt,
			Version:    &ar.Version,
			ParentPath: &pathToTopMostParent,
		},
		Relationships: &app.AreaRelations{
			Space: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &space.SpaceType,
					ID:   &spaceID,
				},
				Links: &app.GenericLinks{
					Self:    &spaceRelatedURL,
					Related: &spaceRelatedURL,
				},
			},
			Children: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Self:    &childURL,
					Related: &childURL,
				},
			},
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}

	// Now check the path, if the path is empty, then this is the topmost area
	// in a specific space.
	if ar.Path.IsEmpty() == false {
		parent := ar.Path.This().String()
		// Only the immediate parent's URL.
		parentSelfURL := rest.AbsoluteURL(request, app.AreaHref(parent))

		i.Relationships.Parent = &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &areaType,
				ID:   &parent,
			},
			Links: &app.GenericLinks{
				Self: &parentSelfURL,
			},
		}
	}
	for _, add := range additional {
		add(appl, request, &ar, i)
	}
	return i
}

// ConvertAreaSimple converts a simple area ID into a Generic Reletionship
func ConvertAreaSimple(request *goa.RequestData, id interface{}) *app.GenericData {
	t := area.APIStringTypeAreas
	i := fmt.Sprint(id)
	return &app.GenericData{
		Type:  &t,
		ID:    &i,
		Links: createAreaLinks(request, id),
	}
}

func createAreaLinks(request *goa.RequestData, id interface{}) *app.GenericLinks {
	relatedURL := rest.AbsoluteURL(request, app.AreaHref(id))
	return &app.GenericLinks{
		Self:    &relatedURL,
		Related: &relatedURL,
	}
}
