package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

// NamedWorkItemsController implements the named_work_items resource.
type NamedWorkItemsController struct {
	*goa.Controller
	db application.DB
}

// NewNamedWorkItemsController creates a named_work_items controller.
func NewNamedWorkItemsController(service *goa.Service, db application.DB) *NamedWorkItemsController {
	return &NamedWorkItemsController{
		Controller: service.NewController("NamedWorkItemsController"),
		db:         db,
	}
}

// Show shows a work item from the given named space (ie, space name along with owner's username) and its number
func (c *NamedWorkItemsController) Show(ctx *app.ShowNamedWorkItemsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		wiID, _, err := appl.WorkItems().LookupIDByNamedSpaceAndNumber(ctx, ctx.UserName, ctx.SpaceName, ctx.WiNumber)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errs.Wrapf(err, "Fail to load work item with number %v in %s/%s", ctx.WiNumber, ctx.UserName, ctx.SpaceName))
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.WorkitemHref(wiID)))
		return ctx.MovedPermanently()
	})
}
