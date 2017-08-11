package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var space = a.Type("Space", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("spaces")
	})
	a.Attribute("id", d.UUID, "ID of the space", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", spaceAttributes)
	a.Attribute("links", genericLinksForSpace)
	a.Required("type", "attributes")
	a.Attribute("relationships", spaceRelationships)
})

var genericLinksForSpace = a.Type("GenericLinksForSpace", func() {
	a.Attribute("self", d.String)
	a.Attribute("related", d.String)
	a.Attribute("backlog", backlogGenericLinkType, `URL to the backlog work items`)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
	a.Attribute("workitemtypes", d.String, "URL to list all WITs for this space")
	a.Attribute("workitemlinktypes", d.String, "URL to list all WILTs for this space")
	a.Attribute("collaborators", d.String, `URL to the list of the space collaborators`)
	a.Attribute("filters", d.String, `URL to the list of available filters`)
})

var backlogGenericLinkType = a.Type("BacklogGenericLink", func() {
	a.Attribute("self", d.String)
	a.Attribute("meta", backlogLinkMeta)
})

var backlogLinkMeta = a.Type("BacklogLinkMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var spaceRelationships = a.Type("SpaceRelationships", func() {
	a.Attribute("owned-by", spaceOwnedBy, "The owner of the Space")
	a.Attribute("iterations", relationGeneric, "Space can have one or many iterations")
	a.Attribute("areas", relationGeneric, "Space can have one or many areas")
	a.Attribute("workitemlinktypes", relationGeneric, "Space can have one or many work item link types")
	a.Attribute("workitemtypes", relationGeneric, "Space can have one or many work item types")
	a.Attribute("workitems", relationGeneric, "Space can have one or many work items")
	a.Attribute("codebases", relationGeneric, "Space can have one or many codebases")
	a.Attribute("collaborators", relationGeneric, `Space can have one or many collaborators`)
})

var spaceOwnedBy = a.Type("SpaceOwnedBy", func() {
	a.Attribute("data", identityRelationData)
	a.Attribute("links", genericLinks)
	a.Required("data")
})

var spaceAttributes = a.Type("SpaceAttributes", func() {
	a.Attribute("name", d.String, "Name for the space", nameValidationFunction)
	a.Attribute("description", d.String, "Description for the space", func() {
		a.Example("This is the foobar collaboration space")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
	a.Attribute("created-at", d.DateTime, "When the space was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the space was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
})

var spaceListMeta = a.Type("SpaceListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var spaceList = JSONList(
	"Space", "Holds the paginated response to a space list request",
	space,
	pagingLinks,
	spaceListMeta)

var spaceSingle = JSONSingle(
	"Space", "Holds a single response to a space request",
	space,
	nil)

// relationSpaces is the JSONAPI store for the spaces
var relationSpaces = a.Type("RelationSpaces", func() {
	a.Attribute("data", relationSpacesData)
	a.Attribute("links", genericLinks)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
})

// relationSpacesData is the JSONAPI data object of the space relationship objects
var relationSpacesData = a.Type("RelationSpacesData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("spaces")
	})
	a.Attribute("id", d.UUID, "UUID for the space", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Attribute("links", genericLinks)
})

var _ = a.Resource("space", func() {
	a.BasePath("/spaces")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:spaceID"),
		)
		a.Description("Retrieve space (as JSONAPI) for the given ID.")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, spaceSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)
		a.Description("List spaces.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, spaceList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a space")
		a.Payload(spaceSingle)
		a.Response(d.Created, "/spaces/.*", func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:spaceID"),
		)
		a.Description("Delete a space with the given ID.")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space to delete")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:spaceID"),
		)
		a.Description("Update the space with the given ID.")
		a.Params(func() {
			a.Param("spaceID", d.UUID, "ID of the space to update")
		})
		a.Payload(spaceSingle)
		a.Response(d.OK, func() {
			a.Media(spaceSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})
