package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// createWorkItemLinkTypePayload defines the structure of work item link type payload in JSONAPI format during creation
var createWorkItemLinkTypePayload = a.Type("CreateWorkItemLinkTypePayload", func() {
	a.Attribute("data", workItemLinkTypeData)
	a.Required("data")
})

// updateWorkItemLinkTypePayload defines the structure of work item link type payload in JSONAPI format during update
var updateWorkItemLinkTypePayload = a.Type("UpdateWorkItemLinkTypePayload", func() {
	a.Attribute("data", workItemLinkTypeData)
	a.Required("data")
})

// workItemLinkTypeListMeta holds meta information for a work item link type array response
var workItemLinkTypeListMeta = a.Type("WorkItemLinkTypeListMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// workItemLinkTypeData is the JSONAPI store for the data of a work item link type.
var workItemLinkTypeData = a.Type("WorkItemLinkTypeData", func() {
	a.Description(`JSONAPI store for the data of a work item link type.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinktypes")
	})
	a.Attribute("id", d.UUID, "ID of work item link type (optional during creation)")
	a.Attribute("attributes", workItemLinkTypeAttributes)
	a.Attribute("relationships", workItemLinkTypeRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

// workItemLinkTypeAttributes is the JSONAPI store for all the "attributes" of a work item link type.
var workItemLinkTypeAttributes = a.Type("WorkItemLinkTypeAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link type.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "Name of the work item link type (required on creation, optional on update)", nameValidationFunction)
	a.Attribute("description", d.String, "Description of the work item link type (optional)", func() {
		a.Example("A test work item can 'test' if a the code in a pull request passes the tests.")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})
	a.Attribute("created-at", d.DateTime, "Time of creation of the given work item type")
	a.Attribute("updated-at", d.DateTime, "Time of last update of the given work item type")
	a.Attribute("forward_name", d.String, `The forward oriented path from source to target is described with the forward name.
For example, if a bug blocks a user story, the forward name is "blocks". See also reverse name.`, func() {
		a.Example("test-workitemtype")
	})
	a.Attribute("reverse_name", d.String, `The backwards oriented path from target to source is described with the reverse name.
For example, if a bug blocks a user story, the reverse name name is "blocked by" as in: a user story is blocked by a bug. See also forward name.`, func() {
		a.Example("tested by")
	})
	a.Attribute("topology", d.String, `The topology determines the restrictions placed on the usage of each work item link type.`, func() {
		a.Enum("network", "tree")
	})

	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link type.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("name")
})

// workItemLinkTypeRelationships is the JSONAPI store for the relationships of a work item link type.
var workItemLinkTypeRelationships = a.Type("WorkItemLinkTypeRelationships", func() {
	a.Description(`JSONAPI store for the data of a work item link type.
See also http://jsonapi.org/format/#document-resource-object-relationships`)
	a.Attribute("link_category", relationWorkItemLinkCategory, "The work item link category of this work item link type.")
	a.Attribute("space", relationSpaces, "This defines the owning space of this work item link type.")
})

// relationWorkItemType is the JSONAPI store for the work item type relationship objects
var relationWorkItemType = a.Type("RelationWorkItemType", func() {
	a.Attribute("data", relationWorkItemTypeData)
	a.Attribute("links", genericLinks)
})

// relationWorkItemTypeData is the JSONAPI data object of the the work item type relationship objects
var relationWorkItemTypeData = a.Type("RelationWorkItemTypeData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("workitemtypes")
	})
	a.Attribute("id", d.UUID, "ID of a work item type")
	a.Required("type", "id")
})

// relationWorkItemLinkType is the JSONAPI store for the links
var relationWorkItemLinkType = a.Type("RelationWorkItemLinkType", func() {
	a.Attribute("data", relationWorkItemLinkTypeData)
})

// relationWorkItemLinkTypeData is the JSONAPI data object of the the work item link type relationship objects
var relationWorkItemLinkTypeData = a.Type("RelationWorkItemLinkTypeData", func() {
	a.Attribute("type", d.String, "The type of the related source", func() {
		a.Enum("workitemlinktypes")
	})
	a.Attribute("id", d.UUID, "ID of work item link type")
	a.Required("type", "id")
})

// workItemLinkTypeLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemLinkTypeLinks = a.Type("WorkItemLinkTypeLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitemlinktypes/2d98c73d-6969-4ea6-958a-812c832b6c18")
	})
	a.Required("self")
})

// ############################################################################
//
//  Media Type Definition
//
// ############################################################################

// workItemLinkType is the media type for work item link types
var workItemLinkType = JSONSingle(
	"WorkItemLinkType",
	`Defines the type of link between two work items.`,
	workItemLinkTypeData,
	workItemLinkTypeLinks,
)

// workItemLinkTypeList contains paged results for listing work item link types and paging links
var workItemLinkTypeList = JSONList(
	"WorkItemLinkType",
	"Holds the paginated response to a work item link type list request",
	workItemLinkTypeData,
	nil, //pagingLinks,
	workItemLinkTypeListMeta,
)

// ############################################################################
//
//  Resource Definition
//
// ############################################################################

var _ = a.Resource("work_item_link_type", func() {
	a.BasePath("/workitemlinktypes")
	a.Parent("space")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:wiltID"),
		)
		a.Description("Retrieve work item link type (as JSONAPI) for the given link ID.")
		a.Params(func() {
			a.Param("wiltID", d.UUID, "ID of the work item link type")
		})
		a.UseTrait("conditional")
		a.Response(d.OK, workItemLinkType)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("list", func() {
		a.Routing(
			a.GET(""),
		)
		a.Description("List work item link types.")
		a.UseTrait("conditional")
		a.Response(d.OK, workItemLinkTypeList)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)
		a.Description("Create a work item link type")
		a.Payload(createWorkItemLinkTypePayload)
		a.Response(d.MethodNotAllowed)
		a.Response(d.Created, "/workitemlinktypes/.*", func() {
			a.Media(workItemLinkType)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:wiltID"),
		)
		a.Description("Delete work item link type with given id.")
		a.Params(func() {
			a.Param("wiltID", d.UUID, "wiltID")
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
			a.PATCH("/:wiltID"),
		)
		a.Description("Update the given work item link type with given id.")
		a.Params(func() {
			a.Param("wiltID", d.UUID, "wiltID")
		})
		a.Payload(updateWorkItemLinkTypePayload)
		a.Response(d.MethodNotAllowed)
		a.Response(d.OK, workItemLinkType)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Conflict, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
