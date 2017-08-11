package controller

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/goadesign/goa"
)

// UserServiceController implements the UserService resource.
type UserServiceController struct {
	*goa.Controller
	UpdateTenant func(context.Context) error
}

// NewUserServiceController creates a UserService controller.
func NewUserServiceController(service *goa.Service) *UserServiceController {
	return &UserServiceController{Controller: service.NewController("UserServiceController")}
}

// Update runs the update action.
func (c *UserServiceController) Update(ctx *app.UpdateUserServiceContext) error {
	c.UpdateTenant(ctx)
	return ctx.OK([]byte{})
}
