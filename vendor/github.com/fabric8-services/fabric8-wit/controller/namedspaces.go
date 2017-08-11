package controller

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
)

// NamedspacesController implements the namedspaces resource.
type NamedspacesController struct {
	*goa.Controller
	db application.DB
}

// NewNamedspacesController creates a namedspaces controller.
func NewNamedspacesController(service *goa.Service, db application.DB) *NamedspacesController {
	return &NamedspacesController{Controller: service.NewController("NamedspacesController"), db: db}
}

// Show runs the show action.
func (c *NamedspacesController) Show(ctx *app.ShowNamedspacesContext) error {
	if ctx.UserName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
	}

	if ctx.SpaceName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, spaceName=%v", ctx.SpaceName))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, ctx.UserName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound("not found, userName=%v", ctx.UserName))
		}
		s, err := appl.Spaces().LoadByOwnerAndName(ctx.Context, &identity.ID, &ctx.SpaceName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		spaceData, err := ConvertSpaceFromModel(ctx.Context, c.db, ctx.RequestData, *s)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		resp := app.SpaceSingle{
			Data: spaceData,
		}

		return ctx.OK(&resp)
	})
}

func (c *NamedspacesController) List(ctx *app.ListNamedspacesContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	if ctx.UserName == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(fmt.Sprintf("not found, userName=%v", ctx.UserName)))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, ctx.UserName)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrNotFound(fmt.Sprintf("not found, userName=%v. %v", ctx.UserName, err.Error())))
		}
		spaces, cnt, err := appl.Spaces().LoadByOwner(ctx.Context, &identity.ID, &offset, &limit)
		count := int(cnt)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		spaceData, err := ConvertSpacesFromModel(ctx.Context, c.db, ctx.RequestData, spaces)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		response := app.SpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  spaceData,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(spaces), offset, limit, count)

		return ctx.OK(&response)
	})
}

func loadKeyCloakIdentityByUserName(ctx context.Context, appl application.Application, username string) (*account.Identity, error) {
	identities, err := appl.Identities().Query(account.IdentityFilterByUsername(username))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"username": username,
		}, "Fail to locate identity for user")
		return nil, err
	}
	for _, identity := range identities {
		if identity.ProviderType == account.KeycloakIDP {
			return &identity, nil
		}
	}
	log.Error(ctx, map[string]interface{}{
		"username": username,
	}, "Fail to locate Keycloak identity for user")
	return nil, fmt.Errorf("Can't find Keycloak Identity for user %s", username)
}
