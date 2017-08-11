package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("namedspaces", func() {
	a.BasePath("/namedspaces")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:userName/:spaceName"),
		)
		a.Description("Retrieve space (as JSONAPI) for the given user name and space name.")
		a.Params(func() {
			a.Param("userName", d.String, "User name of the owner of the space")
			a.Param("spaceName", d.String, "Name of the space, unique to a group of spaces owned by a user")
		})
		a.Response(d.OK, func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET("/:userName"),
		)
		a.Description("List spaces owned by a user.")
		a.Params(func() {
			a.Param("userName", d.String, "User name of the owner of the space")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})

		a.Response(d.OK, func() {
			a.Media(spaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
