package controller

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	// APIStringTypeCodebase contains the JSON API type for codebases
	APIStringTypeSpace = "spaces"
	spaceResourceType  = "space"
)

var scopes = []string{"read:space", "admin:space"}

// SpaceConfiguration represents space configuratoin
type SpaceConfiguration interface {
	GetKeycloakEndpointAuthzResourceset(*goa.RequestData) (string, error)
	GetKeycloakEndpointToken(*goa.RequestData) (string, error)
	GetKeycloakEndpointClients(*goa.RequestData) (string, error)
	GetKeycloakEndpointAdmin(*goa.RequestData) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
	GetCacheControlSpaces() string
}

// SpaceController implements the space resource.
type SpaceController struct {
	*goa.Controller
	db              application.DB
	config          SpaceConfiguration
	resourceManager auth.AuthzResourceManager
}

// NewSpaceController creates a space controller.
func NewSpaceController(service *goa.Service, db application.DB, config SpaceConfiguration, resourceManager auth.AuthzResourceManager) *SpaceController {
	return &SpaceController{Controller: service.NewController("SpaceController"), db: db, config: config, resourceManager: resourceManager}
}

// Create runs the create action.
func (c *SpaceController) Create(ctx *app.CreateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	err = validateCreateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	reqSpace := ctx.Payload.Data
	spaceName := *reqSpace.Attributes.Name
	spaceID := uuid.NewV4()
	if reqSpace.ID != nil {
		spaceID = *reqSpace.ID
	}

	var rSpace *space.Space
	err = application.Transactional(c.db, func(appl application.Application) error {
		newSpace := space.Space{
			ID:      spaceID,
			Name:    spaceName,
			OwnerId: *currentUser,
		}
		if reqSpace.Attributes.Description != nil {
			newSpace.Description = *reqSpace.Attributes.Description
		}

		rSpace, err = appl.Spaces().Create(ctx, &newSpace)
		if err != nil {
			return errs.Wrapf(err, "Failed to create space: %s", newSpace.Name)
		}
		/*
			Should we create the new area
			- over the wire(service) something like app.NewCreateSpaceAreasContext(..), OR
			- as part of a db transaction ?

			The argument 'for' creating it at a transaction level is :
			You absolutely need both space creation + area creation
			to happen in a single transaction as per requirements.
		*/

		newArea := area.Area{
			ID:      uuid.NewV4(),
			SpaceID: rSpace.ID,
			Name:    rSpace.Name,
		}
		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return errs.Wrapf(err, "failed to create area: %s", rSpace.Name)
		}

		// Similar to above, we create a root iteration for this new space
		newIteration := iteration.Iteration{
			ID:      uuid.NewV4(),
			SpaceID: rSpace.ID,
			Name:    rSpace.Name,
		}
		err = appl.Iterations().Create(ctx, &newIteration)
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration for space: %s", rSpace.Name)
		}
		return nil
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	// Create keycloak resource for this space
	resource, err := c.resourceManager.CreateResource(ctx, ctx.RequestData, spaceID.String(), spaceResourceType, &spaceName, &scopes, currentUser.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	spaceResource := &space.Resource{
		ResourceID:   resource.ResourceID,
		PolicyID:     resource.PolicyID,
		PermissionID: resource.PermissionID,
		SpaceID:      spaceID,
	}

	err = application.Transactional(c.db, func(appl application.Application) error {
		// Create space resource which will represent the keyclok resource associated with this space
		_, err = appl.SpaceResources().Create(ctx, spaceResource)
		return err
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	spaceData, err := ConvertSpaceFromModel(ctx.Context, c.db, ctx.RequestData, *rSpace)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	res := &app.SpaceSingle{
		Data: spaceData,
	}
	ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.SpaceHref(res.Data.ID)))
	return ctx.Created(res)
}

// Delete runs the delete action.
func (c *SpaceController) Delete(ctx *app.DeleteSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var resourceID string
	var permissionID string
	var policyID string
	err = application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			return err
		}
		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Warn(ctx, map[string]interface{}{
				"space_id":     ctx.SpaceID,
				"space_owner":  s.OwnerId,
				"current_user": *currentUser,
			}, "user is not the space owner")
			return errors.NewForbiddenError("user is not the space owner")
		}

		// FIXME: what about relying on CASCADE DELETE in the DB instead of doing this ?
		// Delete associated space resource
		resource, err := appl.SpaceResources().LoadBySpace(ctx, &ctx.SpaceID)
		if err != nil {
			return err
		}
		resourceID = resource.ResourceID
		permissionID = resource.PermissionID
		policyID = resource.PolicyID

		appl.SpaceResources().Delete(ctx, resource.ID)
		if err != nil {
			return err
		}
		return appl.Spaces().Delete(ctx.Context, ctx.SpaceID)
	})

	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	c.resourceManager.DeleteResource(ctx, ctx.RequestData, auth.Resource{ResourceID: resourceID, PermissionID: permissionID, PolicyID: policyID})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.OK([]byte{})
}

// List runs the list action.
func (c *SpaceController) List(ctx *app.ListSpaceContext) error {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	var response app.SpaceList
	txnErr := application.Transactional(c.db, func(appl application.Application) error {
		spaces, cnt, err := appl.Spaces().List(ctx.Context, &offset, &limit)
		if err != nil {
			return err
		}
		entityErr := ctx.ConditionalEntities(spaces, c.config.GetCacheControlSpaces, func() error {
			count := int(cnt)
			spaceData, err := ConvertSpacesFromModel(ctx.Context, c.db, ctx.RequestData, spaces)
			if err != nil {
				return err
			}
			response = app.SpaceList{
				Links: &app.PagingLinks{},
				Meta:  &app.SpaceListMeta{TotalCount: count},
				Data:  spaceData,
			}
			setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(spaces), offset, limit, count)
			return nil
		})
		if entityErr != nil {
			return entityErr
		}

		return nil
	})
	if txnErr != nil {
		return jsonapi.JSONErrorResponse(ctx, txnErr)
	}
	return ctx.OK(&response)
}

// Show runs the show action.
func (c *SpaceController) Show(ctx *app.ShowSpaceContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err":      err,
				"space_id": ctx.SpaceID,
			}, "unable to load the space by ID")
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.ConditionalRequest(*s, c.config.GetCacheControlSpaces, func() error {
			spaceData, err := ConvertSpaceFromModel(ctx.Context, c.db, ctx.RequestData, *s)
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"err":      err,
					"space_id": ctx.SpaceID,
				}, "unable to convert the space object")
				return jsonapi.JSONErrorResponse(ctx, err)
			}
			result := &app.SpaceSingle{
				Data: spaceData,
			}
			return ctx.OK(result)
		})
	})
}

// Update runs the update action.
func (c *SpaceController) Update(ctx *app.UpdateSpaceContext) error {
	currentUser, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	err = validateUpdateSpace(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	var response app.SpaceSingle
	txnErr := application.Transactional(c.db, func(appl application.Application) error {
		s, err := appl.Spaces().Load(ctx.Context, ctx.SpaceID)
		if err != nil {
			return err
		}

		if !uuid.Equal(*currentUser, s.OwnerId) {
			log.Error(ctx, map[string]interface{}{"currentUser": *currentUser, "owner": s.OwnerId}, "Current user is not owner")
			return goa.NewErrorClass("forbidden", 403)("User is not the space owner")
		}

		s.Version = *ctx.Payload.Data.Attributes.Version
		if ctx.Payload.Data.Attributes.Name != nil {
			s.Name = *ctx.Payload.Data.Attributes.Name
		}
		if ctx.Payload.Data.Attributes.Description != nil {
			s.Description = *ctx.Payload.Data.Attributes.Description
		}

		s, err = appl.Spaces().Save(ctx.Context, s)
		if err != nil {
			return err
		}

		spaceData, err := ConvertSpaceFromModel(ctx.Context, c.db, ctx.RequestData, *s)
		if err != nil {
			return err
		}
		response = app.SpaceSingle{
			Data: spaceData,
		}
		return nil
	})
	if txnErr != nil {
		return jsonapi.JSONErrorResponse(ctx, txnErr)
	}

	return ctx.OK(&response)
}

func validateCreateSpace(ctx *app.CreateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	return nil
}

func validateUpdateSpace(ctx *app.UpdateSpaceContext) error {
	if ctx.Payload.Data == nil {
		return errors.NewBadParameterError("data", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes == nil {
		return errors.NewBadParameterError("data.attributes", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Name == nil {
		return errors.NewBadParameterError("data.attributes.name", nil).Expected("not nil")
	}
	if ctx.Payload.Data.Attributes.Version == nil {
		return errors.NewBadParameterError("data.attributes.version", nil).Expected("not nil")
	}
	return nil
}

// ConvertSpaceToModel converts an `app.Space` to a `space.Space`
func ConvertSpaceToModel(appSpace app.Space) space.Space {
	modelSpace := space.Space{}

	if appSpace.ID != nil {
		modelSpace.ID = *appSpace.ID
	}
	if appSpace.Attributes != nil {
		if appSpace.Attributes.CreatedAt != nil {
			modelSpace.CreatedAt = *appSpace.Attributes.CreatedAt
		}
		if appSpace.Attributes.UpdatedAt != nil {
			modelSpace.UpdatedAt = *appSpace.Attributes.UpdatedAt
		}
		if appSpace.Attributes.Version != nil {
			modelSpace.Version = *appSpace.Attributes.Version
		}
		if appSpace.Attributes.Name != nil {
			modelSpace.Name = *appSpace.Attributes.Name
		}
		if appSpace.Attributes.Description != nil {
			modelSpace.Description = *appSpace.Attributes.Description
		}
	}
	if appSpace.Relationships != nil && appSpace.Relationships.OwnedBy != nil &&
		appSpace.Relationships.OwnedBy.Data != nil && appSpace.Relationships.OwnedBy.Data.ID != nil {
		modelSpace.OwnerId = *appSpace.Relationships.OwnedBy.Data.ID
	}
	return modelSpace
}

// SpaceConvertFunc is a open ended function to add additional links/data/relations to a Space during
// conversion from internal to API
type SpaceConvertFunc func(*goa.RequestData, *space.Space, *app.Space)

// ConvertSpacesFromModel converts between internal and external REST representation
func ConvertSpacesFromModel(ctx context.Context, db application.DB, request *goa.RequestData, spaces []space.Space, additional ...SpaceConvertFunc) ([]*app.Space, error) {
	var ps = []*app.Space{}
	for _, p := range spaces {
		spaceData, err := ConvertSpaceFromModel(ctx, db, request, p, additional...)
		if err != nil {
			return nil, err
		}

		ps = append(ps, spaceData)
	}
	return ps, nil
}

// ConvertSpaceFromModel converts between internal and external REST representation
func ConvertSpaceFromModel(ctx context.Context, db application.DB, request *goa.RequestData, sp space.Space, additional ...SpaceConvertFunc) (*app.Space, error) {
	relatedURL := rest.AbsoluteURL(request, app.SpaceHref(sp.ID))
	spaceIDStr := sp.ID.String()
	relatedIterationList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/iterations", spaceIDStr))
	relatedAreaList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/areas", spaceIDStr))
	relatedBacklogList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/backlog", spaceIDStr))
	relatedCodebasesList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/codebases", spaceIDStr))
	relatedWorkItemList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/workitems", spaceIDStr))
	relatedWorkItemTypeList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/workitemtypes", spaceIDStr))
	relatedWorkItemLinkTypeList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/workitemlinktypes", spaceIDStr))
	relatedOwnerByLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, sp.OwnerId.String()))
	relatedCollaboratorList := rest.AbsoluteURL(request, fmt.Sprintf("/api/spaces/%s/collaborators", spaceIDStr))
	relatedFilterList := rest.AbsoluteURL(request, "/api/filters")

	count, err := countBacklogItems(ctx, db, sp.ID)
	if err != nil {
		return nil, errs.Wrap(err, "unable to fetch backlog items")
	}
	s := &app.Space{
		ID:   &sp.ID,
		Type: APIStringTypeSpace,
		Attributes: &app.SpaceAttributes{
			Name:        &sp.Name,
			Description: &sp.Description,
			CreatedAt:   &sp.CreatedAt,
			UpdatedAt:   &sp.UpdatedAt,
			Version:     &sp.Version,
		},
		Links: &app.GenericLinksForSpace{
			Self:    &relatedURL,
			Related: &relatedURL,
			Backlog: &app.BacklogGenericLink{
				Self: &relatedBacklogList,
				Meta: &app.BacklogLinkMeta{TotalCount: count},
			},
			Workitemtypes:     &relatedWorkItemTypeList,
			Workitemlinktypes: &relatedWorkItemLinkTypeList,
			Filters:           &relatedFilterList,
		},
		Relationships: &app.SpaceRelationships{
			OwnedBy: &app.SpaceOwnedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &sp.OwnerId,
				},
				Links: &app.GenericLinks{
					Related: &relatedOwnerByLink,
				},
			},
			Iterations: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedIterationList,
				},
			},
			Areas: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedAreaList,
				},
			},
			Codebases: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedCodebasesList,
				},
			},
			Workitems: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedWorkItemList,
				},
			},
			Collaborators: &app.RelationGeneric{
				Links: &app.GenericLinks{
					Related: &relatedCollaboratorList,
				},
			},
		},
	}
	for _, add := range additional {
		add(request, &sp, s)
	}
	return s, nil
}
