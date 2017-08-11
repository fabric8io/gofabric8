package controller

import (
	"fmt"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	"github.com/pkg/errors"
)

// UserController implements the user resource.
type UserController struct {
	*goa.Controller
	db           application.DB
	tokenManager token.Manager
	config       UserControllerConfiguration
	InitTenant   func(context.Context) error
}

// UserControllerConfiguration the configuration for the UserController
type UserControllerConfiguration interface {
	GetCacheControlUser() string
}

// NewUserController creates a user controller.
func NewUserController(service *goa.Service, db application.DB, tokenManager token.Manager, config UserControllerConfiguration) *UserController {
	return &UserController{
		Controller:   service.NewController("UserController"),
		db:           db,
		tokenManager: tokenManager,
		config:       config,
	}
}

// Show returns the authorized user based on the provided Token
func (c *UserController) Show(ctx *app.ShowUserContext) error {
	id, err := c.tokenManager.Locate(ctx)
	if err != nil {
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest(err.Error()))
		return ctx.BadRequest(jerrors)
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		identity, err := appl.Identities().Load(ctx, id)
		if err != nil || identity == nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": id,
			}, "auth token containers id %s of unknown Identity", id)
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrUnauthorized(fmt.Sprintf("Auth token contains id %s of unknown Identity\n", id)))
			return ctx.Unauthorized(jerrors)
		}
		var user *account.User
		userID := identity.UserID
		if userID.Valid {
			user, err = appl.Users().Load(ctx.Context, userID.UUID)
			if err != nil {
				return jsonapi.JSONErrorResponse(ctx, errors.Wrap(err, fmt.Sprintf("Can't load user with id %s", userID.UUID)))
			}
		}
		return ctx.ConditionalRequest(*user, c.config.GetCacheControlUser, func() error {
			if c.InitTenant != nil {
				go func(ctx context.Context) {
					c.InitTenant(ctx)
				}(ctx)
			}
			return ctx.OK(ConvertToAppUser(ctx.RequestData, user, identity))
		})
	})
}
