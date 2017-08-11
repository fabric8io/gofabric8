package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var searchWorkItemList = JSONList(
	"SearchWorkItem", "Holds the paginated response to a search request",
	workItem,
	pagingLinks,
	meta)

var searchSpaceList = JSONList(
	"SearchSpace", "Holds the paginated response to a search request",
	space,
	pagingLinks,
	spaceListMeta)

var _ = a.Resource("search", func() {
	a.BasePath("/search")

	a.Action("show", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("Search by ID, URL, full text capability")
		a.Params(func() {
			a.Param("q", d.String,
				`Following are valid input for search query
				1) "id:100" :- Look for work item hainvg id 100
				2) "url:http://demo.openshift.io/details/500" :- Search on WI having id 500 and check 
					if this URL is mentioned in searchable columns of work item
				3) "simple keywords separated by space" :- Search in Work Items based on these keywords.`)
			a.Param("page[offset]", d.String, "Paging start position") // #428
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Param("filter[expression]", d.String, "Filter expression in JSON format", func() {
				a.Example(`{$AND: [{"space": "f73988a2-1916-4572-910b-2df23df4dcc3"}, {"state": "NEW"}]}`)
			})
			a.Param("spaceID", d.String, "The optional space ID of the space to be searched in")
		})
		a.Response(d.OK, func() {
			a.Media(searchWorkItemList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("spaces", func() {
		a.Routing(
			a.GET("spaces"),
		)
		a.Description("Search for spaces by name or description")
		a.Params(func() {
			a.Param("q", d.String, "Text to match against Name or description")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Required("q")
		})
		a.Response(d.OK, func() {
			a.Media(searchSpaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("users", func() {
		a.Routing(
			a.GET("users"),
		)
		a.Description("Search by fullname")
		a.Params(func() {
			a.Param("q", d.String)
			a.Param("page[offset]", d.String, "Paging start position") // #428
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Required("q")
		})
		a.Response(d.OK, func() {
			a.Media(userList)
		})

		a.Response(d.BadRequest, func() {
			a.Media(d.ErrorMedia)
		})

		a.Response(d.InternalServerError)
	})
})
