package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// workItemLinkCategoryLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemLinkCategoryLinks = a.Type("WorkItemLinkCategoryLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitemlinkcategories/2d98c73d-6969-4ea6-958a-812c832b6c18")
	})
	a.Required("self")
})

// createWorkItemLinkCategoryPayload defines the structure of work item link category payload in JSONAPI format during creation
var createWorkItemLinkCategoryPayload = a.Type("CreateWorkItemLinkCategoryPayload", func() {
	a.Attribute("data", workItemLinkCategoryData)
	a.Required("data")
})

// updateWorkItemLinkCategoryPayload defines the structure of work item link category payload in JSONAPI format during update
var updateWorkItemLinkCategoryPayload = a.Type("UpdateWorkItemLinkCategoryPayload", func() {
	a.Attribute("data", workItemLinkCategoryData)
	a.Required("data")
})

// workItemLinkCategoryListMeta holds meta information for a work item link category array response
var workItemLinkCategoryListMeta = a.Type("WorkItemLinkCategoryListMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// workItemLinkCategoryData is the JSONAPI store for the data of a work item link category.
var workItemLinkCategoryData = a.Type("WorkItemLinkCategoryData", func() {
	a.Description(`JSONAPI store the data of a work item link category.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinkcategories")
	})
	a.Attribute("id", d.UUID, "ID of work item link category (optional during creation)", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Attribute("attributes", workItemLinkCategoryAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

// workItemLinkCategoryAttributes is the JSONAPI store for all the "attributes" of a work item link category.
var workItemLinkCategoryAttributes = a.Type("WorkItemLinkCategoryAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link category.
See also http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "Name of the work item link category (required on creation, optional on update)", nameValidationFunction)
	a.Attribute("description", d.String, "Description of the work item link category (optional)", func() {
		a.Example("A work item link category that is meant only for work item link types goverened by the system alone.")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})

	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link category.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("name")
})

// relationWorkItemLinkCategory is the JSONAPI store for the links
var relationWorkItemLinkCategory = a.Type("RelationWorkItemLinkCategory", func() {
	a.Attribute("data", relationWorkItemLinkCategoryData)
	a.Attribute("links", genericLinks)
})

// relationWorkItemLinkCategoryData is the JSONAPI data object of the the work item link category relationship objects
var relationWorkItemLinkCategoryData = a.Type("RelationWorkItemLinkCategoryData", func() {
	a.Attribute("type", d.String, "The type of the related source", func() {
		a.Enum("workitemlinkcategories")
	})
	a.Attribute("id", d.UUID, "ID of work item link category")
	a.Required("type", "id")
})

// ############################################################################
//
//  Media Type Definition
//
// ############################################################################

// workItemLinkCategory is the media type for work item link categories
var workItemLinkCategory = JSONSingle(
	"WorkItemLinkCategory",
	`WorkItemLinkCategory puts a category on a link between two work items.
The category is attached to a work item link type. A link type can have a
category like "system", "extension", or "user". Those categories are handled
by this media type.`,
	workItemLinkCategoryData,
	workItemLinkCategoryLinks,
)

// workItemLinkCategoryList contains paged results for listing work item link categories and paging links
var workItemLinkCategoryList = JSONList(
	"WorkItemLinkCategory",
	"Holds the paginated response to a work item link category list request",
	workItemLinkCategoryData,
	nil, //pagingLinks,
	workItemLinkCategoryListMeta,
)

// ############################################################################
//
//  Resource Definition
//
// ############################################################################

var _ = a.Resource("work_item_link_category", func() {
	a.BasePath("/workitemlinkcategories")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:id"),
		)
		a.Description("Retrieve work item link category (as JSONAPI) for the given ID.")
		a.Params(func() {
			a.Param("id", d.UUID, "ID of the work item link category")
		})
		a.Response(d.OK, func() {
			a.Media(workItemLinkCategory)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item link categories.")
		a.Response(d.OK, func() {
			a.Media(workItemLinkCategoryList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a work item link category")
		a.Payload(createWorkItemLinkCategoryPayload)
		a.Response(d.MethodNotAllowed)
		a.Response(d.Created, "/workitemlinkcategories/.*", func() {
			a.Media(workItemLinkCategory)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:id"),
		)
		a.Description("Delete work item link category with given id.")
		a.Params(func() {
			a.Param("id", d.UUID, "id")
		})
		a.Response(d.MethodNotAllowed)
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:id"),
		)
		a.Description("Update the given work item link category with given id.")
		a.Params(func() {
			a.Param("id", d.UUID, "id")
		})
		a.Payload(updateWorkItemLinkCategoryPayload)
		a.Response(d.MethodNotAllowed)
		a.Response(d.OK, func() {
			a.Media(workItemLinkCategory)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
