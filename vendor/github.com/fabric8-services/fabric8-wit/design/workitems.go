package design

import (
	"strings"

	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// genericLinksForWorkItem defines generic relations links that are specific to a workitem
var genericLinksForWorkItem = a.Type("GenericLinksForWorkItem", func() {
	a.Attribute("self", d.String)
	a.Attribute("related", d.String)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
	a.Attribute("editCodebase", d.String, "URL to generate Che-editor's link based on values of codebase field")
})

// workItem defines how an update payload will look like
var workItem = a.Type("WorkItem", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitems")
	})
	a.Attribute("id", d.UUID, "ID of the work item which is being updated", func() {
		a.Example("abcd1234-1234-5678-cafe-0123456789ab")
	})
	a.Attribute("attributes", a.HashOf(d.String, d.Any), func() {
		a.Example(map[string]interface{}{"version": "1", "system.state": "new", "system.title": "Example story"})
	})
	a.Attribute("relationships", workItemRelationships)
	a.Attribute("links", genericLinksForWorkItem)
	a.Required("type", "attributes")
})

// WorkItemRelationships defines only `assignee` as of now. To be updated
var workItemRelationships = a.Type("WorkItemRelationships", func() {
	a.Attribute("assignees", relationGenericList, "This defines assignees of the Work Item")
	a.Attribute("creator", relationGeneric, "This defines creator of the Work Item")
	a.Attribute("baseType", relationBaseType, "This defines type of Work Item")
	a.Attribute("comments", relationGeneric, "This defines comments on the Work Item")
	a.Attribute("iteration", relationGeneric, "This defines the iteration this work item belong to")
	a.Attribute("area", relationGeneric, "This defines the area this work item belongs to")
	a.Attribute("children", relationGeneric, "This defines the children of this work item")
	a.Attribute("space", relationSpaces, "This defines the owning space of this work item.")
})

// relationBaseType is top level block for WorkItemType relationship
var relationBaseType = a.Type("RelationBaseType", func() {
	a.Attribute("data", baseTypeData)
	a.Attribute("links", genericLinks)
	a.Required("data")
})

// baseTypeData is data block for `type` of a work item
var baseTypeData = a.Type("BaseTypeData", func() {
	a.Attribute("type", d.String, func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.UUID, "ID of the work item type")
	a.Required("type", "id")
})

// workItemLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemLinks = a.Type("WorkItemLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitems.2/1")
	})
	a.Required("self")
})

// workItemList contains paged results for listing work items and paging links
var workItemList = JSONList(
	"WorkItem", "Holds the paginated response to a work item list request",
	workItem,
	pagingLinks,
	meta)

// workItemSingle is the media type for work items
var workItemSingle = JSONSingle(
	"WorkItem", "A work item holds field values according to a given field type in JSONAPI form",
	workItem,
	workItemLinks)

// Reorder creates a UserTypeDefinition for Reorder action
func Reorder(name, description string, data *d.UserTypeDefinition, position *d.UserTypeDefinition) *d.MediaTypeDefinition {
	return a.MediaType("application/vnd."+strings.ToLower(name)+"json", func() {
		a.UseTrait("jsonapi-media-type")
		a.TypeName(name + "Reorder")
		a.Description(description)
		a.Attribute("data", a.ArrayOf(data))
		a.Attribute("position", position)
		a.View("default", func() {
			a.Attribute("data")
			a.Required("data")
		})
	})
}

// workItemReorder is the media type for reorder of work items
var workItemReorder = Reorder(
	"WorkItem", "Holds values for work item reorder",
	workItem,
	position)

// endpoints that DO NOT depend on the space id (ie, when the work item ID is specified in the URL, there's no need to pass the space ID)
var _ = a.Resource("workitem", func() {
	a.BasePath("/workitems")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:wiID"),
		)
		a.Description("Retrieve a work item from the given id.")
		a.Params(func() {
			a.Param("wiID", d.UUID, "ID of the work item to show")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list-children", func() {
		a.Routing(
			a.GET("/:wiID/children"),
		)
		a.Description("List children associated with the given work item")
		a.Params(func() {
			a.Param("wiID", d.UUID, "ID of the work item to look-up")
			a.Param("page[offset]", d.String, `Paging start position is a string pointing to the beginning of pagination.  The value starts from 0 onwards.`)
			a.Param("page[limit]", d.Integer, `Paging size is the number of items in a page`)
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:wiID"),
		)
		a.Description("Delete work item with given its id.")
		a.Params(func() {
			a.Param("wiID", d.UUID, "ID of the work item to delete")
		})
		a.Response(d.MethodNotAllowed)
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
			a.PATCH("/:wiID"),
		)
		a.Description("update the work item with the given natural id.")
		a.Params(func() {
			a.Param("wiID", d.UUID, "ID of the work item to update")
		})
		a.Payload(workItemSingle)
		a.Response(d.OK, func() {
			a.Media(workItemSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})

// endpoints that depend on the space id
var _ = a.Resource("workitems", func() {
	a.Parent("space")
	a.BasePath("/workitems")
	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work items.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found work items")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Param("filter[assignee]", d.String, "Work Items assigned to the given user")
			a.Param("filter[iteration]", d.String, "IterationID to filter work items")
			a.Param("filter[workitemtype]", d.UUID, "ID of work item type to filter work items by")
			a.Param("filter[area]", d.String, "AreaID to filter work items")
			a.Param("filter[workitemstate]", d.String, "work item state to filter work items by")
			a.Param("filter[parentexists]", d.Boolean, "if false list work items without any parent")
			a.Param("filter[expression]", d.String, "accepts query in JSON format and redirects to /api/search? API", func() {
				a.Example(`{$AND: [{"space": "f73988a2-1916-4572-910b-2df23df4dcc3"}, {"state": "NEW"}]}`)
			})
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.TemporaryRedirect)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("create work item with type and id.")
		a.Payload(workItemSingle)
		a.Response(d.Created, "/workitems/.*", func() {
			a.Media(workItemSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("reorder", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/reorder"),
		)
		a.Description("reorder the work items")
		a.Payload(workItemReorder)
		a.Response(d.OK, func() {
			a.Media(workItemReorder)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})

var _ = a.Resource("planner_backlog", func() {
	a.Parent("space")
	a.BasePath("/backlog")

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List backlog work items.")
		a.Params(func() {
			a.Param("filter", d.String, "a query language expression restricting the set of found work items")
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
			a.Param("filter[assignee]", d.String, "Work Items assigned to the given user")
			a.Param("filter[workitemtype]", d.UUID, "ID of work item type to filter work items by")
			a.Param("filter[area]", d.String, "AreaID to filter work items")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})

var _ = a.Resource("named_work_items", func() {
	a.Parent("namedspaces")
	a.BasePath("/workitems")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:wiNumber"),
		)
		a.Description("Retrieve a work item from the given number.")
		a.Params(func() {
			a.Param("wiNumber", d.Integer, "Number of the work item to show")
		})
		a.Response(d.MovedPermanently)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
