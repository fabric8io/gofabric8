package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

// WITStatus defines the status of the current running WIT instance
var WITStatus = a.MediaType("application/vnd.status+json", func() {
	a.Description("The status of the current running instance")
	a.Attributes(func() {
		a.Attribute("commit", d.String, "Commit SHA this build is based on")
		a.Attribute("buildTime", d.String, "The time when built")
		a.Attribute("startTime", d.String, "The time when started")
		a.Attribute("error", d.String, "The error if any")
		a.Required("commit", "buildTime", "startTime")
	})
	a.View("default", func() {
		a.Attribute("commit")
		a.Attribute("buildTime")
		a.Attribute("startTime")
		a.Attribute("error")
	})
})

var pagingLinks = a.Type("pagingLinks", func() {
	a.Attribute("prev", d.String)
	a.Attribute("next", d.String)
	a.Attribute("first", d.String)
	a.Attribute("last", d.String)
	a.Attribute("filters", d.String)
})

var meta = a.Type("workItemListResponseMeta", func() {
	a.Attribute("totalCount", d.Integer)

	a.Required("totalCount")
})

// position represents the ID of the workitem above which the to-be-reordered workitem(s) should be placed
var position = a.Type("workItemReorderPosition", func() {
	a.Description("Position represents the ID of the workitem above which the to-be-reordered workitem(s) should be placed")
	a.Attribute("id", d.UUID, "ID of the workitem above which the to-be-reordered workitem(s) should be placed")
	a.Attribute("direction", d.String, "Direction of the place of the reorder workitem. Above should be used to place the reorder workitem(s) above workitem with id equal to position.id. Below should be used to place the reorder workitem(s) below workitem with id equal to position.id. Top places the reorder workitem(s) at the Topmost position of the list. Bottom places the reorder item(s) at the bottom of the list.", func() {
		a.Enum("above", "below", "top", "bottom")
	})

	a.Required("direction")
})

// Tracker configuration
var Tracker = a.MediaType("application/vnd.tracker+json", func() {
	a.TypeName("Tracker")
	a.Description("Tracker configuration")
	a.Attribute("id", d.String, "unique id per tracker")
	a.Attribute("url", d.String, "URL of the tracker")
	a.Attribute("type", d.String, "Type of the tracker")

	a.Required("id")
	a.Required("url")
	a.Required("type")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("url")
		a.Attribute("type")
	})
})

// TrackerQuery represents the search query with schedule
var TrackerQuery = a.MediaType("application/vnd.trackerquery+json", func() {
	a.TypeName("TrackerQuery")
	a.Description("Tracker query with schedule")
	a.Attribute("id", d.String, "unique id per installation")
	a.Attribute("query", d.String, "Search query")
	a.Attribute("schedule", d.String, "Schedule for fetch and import")
	a.Attribute("trackerID", d.String, "Tracker ID")
	a.Attribute("relationships", trackerQueryRelationships)

	a.Required("id")
	a.Required("query")
	a.Required("schedule")
	a.Required("trackerID")
	a.Required("relationships")

	a.View("default", func() {
		a.Attribute("id")
		a.Attribute("query")
		a.Attribute("schedule")
		a.Attribute("trackerID")
		a.Attribute("relationships")
	})
})

var trackerQueryRelationships = a.Type("TrackerQueryRelationships", func() {
	a.Attribute("space", relationSpaces, "This defines the owning space of this work item type.")
})
