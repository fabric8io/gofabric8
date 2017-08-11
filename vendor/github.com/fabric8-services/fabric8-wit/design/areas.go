package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var area = a.Type("Area", func() {
	a.Description(`JSONAPI store for the data of a Area. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("areas")
	})
	a.Attribute("id", d.UUID, "ID of area", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", areaAttributes)
	a.Attribute("relationships", areaRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var areaAttributes = a.Type("AreaAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Area. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The Area name", nameValidationFunction)
	a.Attribute("created-at", d.DateTime, "When the area was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the area was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(23)
	})
	a.Attribute("parent_path", d.String, "Path to the topmost parent", func() {
		a.Example("/40bbdd3d-8b5d-4fd6-ac90-7236b669af04/40bbdd3d-8b5d-4fd6-ac90-7236b669af02")
	})
	a.Attribute("parent_path_resolved", d.String, "Path to the topmost area specified by area names", func() {
		a.Example("/devtools/planner/planner-ui")
	})
})

var areaRelationships = a.Type("AreaRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
	a.Attribute("parent", relationGeneric, "This defines the parents' hierarchy for areas")
	a.Attribute("children", relationGeneric, "This defines the sub-areas present for this area")
	a.Attribute("workitems", relationGeneric, "This defines the workitems associated with the Area")
})

var areaList = JSONList(
	"Area", "Holds the list of Areas",
	area,
	pagingLinks,
	meta)

var areaSingle = JSONSingle(
	"Area", "Holds a single Area",
	area,
	nil)

// new version of "list" for migration
var _ = a.Resource("area", func() {
	a.BasePath("/areas")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve area with given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, areaSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("show-children", func() {
		a.Routing(
			a.GET("/:id/children"),
		)
		a.Description("Retrieve child areas of given id.")
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, areaList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("create-child", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:id"),
		)
		a.Params(func() {
			a.Param("id", d.String, "id")
		})
		a.Description("create child area.")
		a.Payload(areaSingle)
		a.Response(d.Created, "/areas/.*", func() {
			a.Media(areaSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
	})
})

// new version of "list" for migration
var _ = a.Resource("space_areas", func() {
	a.Parent("space")

	a.Action("list", func() {
		a.Routing(
			a.GET("areas"),
		)
		a.Description("List Areas.")
		a.UseTrait("conditional")
		a.Response(d.OK, areaList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
