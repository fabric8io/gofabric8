package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// MarkupRenderingPayload wraps the data in a JSONAPI compliant request
var markupRenderingPayload = a.Type("MarkupRenderingPayload", func() {
	a.Description("A MarkupRenderingPayload describes the values that a render request can hold.")
	a.Attribute("data", markupRenderingPayloadData)
	a.Required("data")
})

// MarkupRenderingPayloadData is the media type representing a rendering input.
var markupRenderingPayloadData = a.Type("MarkupRenderingPayloadData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("rendering")
	})
	a.Attribute("attributes", markupRenderingPayloadDataAttributes)
	a.Required("type")
	a.Required("attributes")
})

// MarkupRenderingPayloadData is the media type representing a rendering input.
var markupRenderingPayloadDataAttributes = a.Type("MarkupRenderingPayloadDataAttributes", func() {
	a.Attribute("content", d.String, "The content to render", func() {
		a.Example("# foo")
	})
	a.Attribute("markup", d.String, "The markup language associated with the content to render", func() {
		a.Example("Markdown")
	})
	a.Required("content")
	a.Required("markup")
})

// MarkupRenderingMediaType is the media type for rendering result
var markupRenderingMediaType = JSONSingle(
	"MarkupRendering",
	`MarkupRenderingMediaType contains the  
		rendering of the 'content' provided in the request, using
		the markup language specified by the 'markup' value.`,
	markupRenderingMediaTypeData,
	nil,
)

// MarkupRenderingMediaType is the data included in the rendering result response.
var markupRenderingMediaTypeData = a.Type("MarkupRenderingData", func() {
	a.Attribute("id", d.String, "an ID to conform to the JSON-API spec, even though it is meaningless in the case of the rendering endpoint. Can be null", func() {
		a.Example("42")
	})
	a.Attribute("type", d.String, func() {
		a.Enum("rendering")
	})
	a.Attribute("attributes", markupRenderingMediaTypeDataAttributes)
	a.Required("id")
	a.Required("type")
	a.Required("attributes")
})

// MarkupRenderingMediaType is the data included in the rendering result response.
var markupRenderingMediaTypeDataAttributes = a.Type("MarkupRenderingDataAttributes", func() {
	a.Attribute("renderedContent", d.String, "The rendered content", func() {
		a.Example("<h1>foo</h1>")
	})
	a.Required("renderedContent")
})

var _ = a.Resource("render", func() {
	a.BasePath("/render")
	a.Security("jwt")
	a.Action("render", func() {
		a.Description("Render some content using the markup language")
		a.Routing(a.POST(""))
		a.Payload(markupRenderingPayload)
		a.Response(d.OK, markupRenderingMediaType)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
