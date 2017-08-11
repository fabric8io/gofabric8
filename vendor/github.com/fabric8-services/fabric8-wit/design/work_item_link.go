package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// workItemLinkLinks has `self` as of now according to http://jsonapi.org/format/#fetching-resources
var workItemLinkLinks = a.Type("WorkItemLinkLinks", func() {
	a.Attribute("self", d.String, func() {
		a.Example("http://api.openshift.io/api/workitemlinks/2d98c73d-6969-4ea6-958a-812c832b6c18")
	})
	a.Required("self")
})

// createWorkItemLinkPayload defines the structure of work item link payload in JSONAPI format during creation
var createWorkItemLinkPayload = a.Type("CreateWorkItemLinkPayload", func() {
	a.Attribute("data", workItemLinkData)
	a.Required("data")
})

// updateWorkItemLinkPayload defines the structure of work item link payload in JSONAPI format during update
var updateWorkItemLinkPayload = a.Type("UpdateWorkItemLinkPayload", func() {
	a.Attribute("data", workItemLinkData)
	a.Required("data")
})

// workItemLinkListMeta holds meta information for a work item link array response
var workItemLinkListMeta = a.Type("WorkItemLinkListMeta", func() {
	a.Attribute("totalCount", d.Integer, func() {
		a.Minimum(0)
	})
	a.Required("totalCount")
})

// workItemLinkData is the JSONAPI store for the data of a work item link.
var workItemLinkData = a.Type("WorkItemLinkData", func() {
	a.Description(`JSONAPI store for the data of a work item.
See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workitemlinks")
	})
	a.Attribute("id", d.UUID, "ID of work item link (optional during creation)")
	a.Attribute("attributes", workItemLinkAttributes)
	a.Attribute("relationships", workItemLinkRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type", "relationships")
})

// workItemLinkAttributes is the JSONAPI store for all the "attributes" of a work item link type.
var workItemLinkAttributes = a.Type("WorkItemLinkAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a work item link.
See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("created-at", d.DateTime, "When the space was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the space was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("version", d.Integer, "Version for optimistic concurrency control (optional during creating)", func() {
		a.Example(0)
	})

	// IMPORTANT: We cannot require any field here because these "attributes" will be used
	// during the creation as well as the update of a work item link type.
	// During creation, the "name" field is required but not during update.
	// The repository methods need to check for required fields.
	//a.Required("version")
})

// workItemLinkRelationships is the JSONAPI store for the relationships of a work item link.
var workItemLinkRelationships = a.Type("WorkItemLinkRelationships", func() {
	a.Description(`JSONAPI store for the data of a work item link.
See also http://jsonapi.org/format/#document-resource-object-relationships`)
	a.Attribute("link_type", relationWorkItemLinkType, "The work item link type of this work item link.")
	a.Attribute("source", relationWorkItem, "Work item where the connection starts.")
	a.Attribute("target", relationWorkItem, "Work item where the connection ends.")
})

// relationWorkItem is the JSONAPI store for the links
var relationWorkItem = a.Type("RelationWorkItem", func() {
	a.Attribute("data", relationWorkItemData)
})

// relationWorkItemData is the JSONAPI data object of the the work item relationship objects
var relationWorkItemData = a.Type("RelationWorkItemData", func() {
	a.Attribute("type", d.String, "The type of the related resource", func() {
		a.Enum("workitems")
	})
	a.Attribute("id", d.UUID, "ID (UUID) of the work item", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Required("type", "id")
})

// ############################################################################
//
//  Media Type Definition
//
// ############################################################################

// workItemLink is the media type for work item links
var workItemLink = JSONSingle(
	"WorkItemLink",
	"Defines a connection between two work items",
	workItemLinkData,
	workItemLinkLinks,
)

// workItemLinkList contains paged results for listing work item links and paging links
var workItemLinkList = JSONList(
	"WorkItemLink",
	"Holds the paginated response to a work item link list request",
	workItemLinkData,
	nil, //pagingLinks,
	workItemLinkListMeta,
)

// ############################################################################
//
//  Resource Definition
//
// ############################################################################

var _ = a.Resource("work_item_link", func() {
	a.BasePath("/workitemlinks")
	a.Action("show", showWorkItemLink)
	a.Action("create", createWorkItemLink)
	a.Action("delete", deleteWorkItemLink)
	a.Action("update", updateWorkItemLink)
})

var _ = a.Resource("work_item_relationships_links", func() {
	a.BasePath("/relationships/links")
	a.Parent("workitem")
	a.Action("list", func() {
		listWorkItemLinks()
		a.Description("List work item links associated with the given work item (either as source or as target work item).")
		a.Response(d.NotFound, JSONAPIErrors, func() {
			a.Description("This error arises when the given work item does not exist.")
		})
	})
	a.Action("create", createWorkItemLink)
})

// listWorkItemLinks defines the list action for endpoints that return an array
// of work item links.
func listWorkItemLinks() {
	a.Description("Retrieve work item link (as JSONAPI) for the given link ID.")
	a.Routing(
		a.GET(""),
	)
	a.UseTrait("conditional")
	a.Response(d.OK, workItemLinkList)
	a.Response(d.NotModified)
	a.Response(d.BadRequest, JSONAPIErrors)
	a.Response(d.InternalServerError, JSONAPIErrors)
}

func showWorkItemLink() {
	a.Description("Retrieve work item link (as JSONAPI) for the given link ID.")
	a.Routing(
		a.GET("/:linkId"),
	)
	a.Params(func() {
		a.Param("linkId", d.UUID, "ID of the work item link to show")
	})
	a.UseTrait("conditional")
	a.Response(d.OK, workItemLink)
	a.Response(d.NotModified)
	a.Response(d.BadRequest, JSONAPIErrors)
	a.Response(d.InternalServerError, JSONAPIErrors)
	a.Response(d.NotFound, JSONAPIErrors)
}

func createWorkItemLink() {
	a.Description("Create a work item link")
	a.Security("jwt")
	a.Routing(
		a.POST(""),
	)
	a.Payload(createWorkItemLinkPayload)
	a.Response(d.Created, "/workitemlinks/.*", func() {
		a.Media(workItemLink)
	})
	a.Response(d.BadRequest, JSONAPIErrors)
	a.Response(d.InternalServerError, JSONAPIErrors)
	a.Response(d.Unauthorized, JSONAPIErrors)
	a.Response(d.NotFound, JSONAPIErrors)
}

func deleteWorkItemLink() {
	a.Description("Delete work item link with given id.")
	a.Security("jwt")
	a.Routing(
		a.DELETE("/:linkId"),
	)
	a.Params(func() {
		a.Param("linkId", d.UUID, "ID of the work item link to be deleted")
	})
	a.Response(d.OK)
	a.Response(d.BadRequest, JSONAPIErrors)
	a.Response(d.InternalServerError, JSONAPIErrors)
	a.Response(d.NotFound, JSONAPIErrors)
	a.Response(d.Unauthorized, JSONAPIErrors)
}

func updateWorkItemLink() {
	a.Description("Update the given work item link with given id.")
	a.Security("jwt")
	a.Routing(
		a.PATCH("/:linkId"),
	)
	a.Params(func() {
		a.Param("linkId", d.UUID, "ID of the work item link to be updated")
	})
	a.Payload(updateWorkItemLinkPayload)
	a.Response(d.OK, func() {
		a.Media(workItemLink)
	})
	a.Response(d.BadRequest, JSONAPIErrors)
	a.Response(d.Conflict, JSONAPIErrors)
	a.Response(d.InternalServerError, JSONAPIErrors)
	a.Response(d.NotFound, JSONAPIErrors)
	a.Response(d.Unauthorized, JSONAPIErrors)
}
