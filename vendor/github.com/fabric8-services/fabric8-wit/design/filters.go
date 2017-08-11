package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var filter = a.Type("filters", func() {
	a.Description(`JSONAPI store for the data of a filter. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("filters")
	})
	a.Attribute("attributes", filterAttributes)
	a.Required("type", "attributes")
})

var filterAttributes = a.Type("filterAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a filter. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("title", d.String, "The Filter name", func() {
		a.Example("Assignee")
	})
	a.Attribute("description", d.String, "When the filter was created", func() {
		a.Example("Filter by assignee")
	})
	a.Attribute("query", d.String, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example("filter[assignee]={id}")
	})
	a.Attribute("type", d.String, "Path to the topmost parent", func() {
		a.Example("users")
	})
	a.Required("type", "title", "description", "query")
})

var filterList = JSONList(
	"filter", "Holds the list of Filters",
	filter,
	pagingLinks, // pagingLinks would eventually remain nil.
	meta)        // again, this being a pointer gets auto-assigned nil.

var filterSingle = JSONSingle(
	"filter", "Holds filter information",
	filter,
	nil)

var _ = a.Resource("filter", func() {
	a.BasePath("/filters")
	a.CanonicalActionName("list")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List filters.")
		a.Response(d.OK, func() {
			a.Media(filterList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
