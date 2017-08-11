package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

const (
	RenderingType = "rendering"
	RenderedValue = "value"
)

// RenderController implements the render resource.
type RenderController struct {
	*goa.Controller
}

// NewRenderController creates a render controller.
func NewRenderController(service *goa.Service) *RenderController {
	return &RenderController{Controller: service.NewController("RenderController")}
}

// Render runs the render action.
func (c *RenderController) Render(ctx *app.RenderRenderContext) error {
	content := ctx.Payload.Data.Attributes.Content
	markup := ctx.Payload.Data.Attributes.Markup
	if !rendering.IsMarkupSupported(markup) {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Unsupported markup type", markup))
	}
	htmlResult := rendering.RenderMarkupToHTML(content, markup)
	res := &app.MarkupRenderingSingle{Data: &app.MarkupRenderingData{
		ID:   uuid.NewV4().String(),
		Type: RenderingType,
		Attributes: &app.MarkupRenderingDataAttributes{
			RenderedContent: htmlResult,
		}}}
	return ctx.OK(res)
}
