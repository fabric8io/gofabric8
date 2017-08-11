package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var codebase = a.Type("Codebase", func() {
	a.Description(`JSONAPI store for the data of a codebase.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("codebases")
	})
	a.Attribute("id", d.UUID, "ID of codebase", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", codebaseAttributes)
	a.Attribute("relationships", codebaseRelationships)
	a.Attribute("links", codebaseLinks)
	a.Required("type", "attributes")
})

var codebaseAttributes = a.Type("CodebaseAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a codebase. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("type", d.String, "The codebase type", func() {
		a.Example("git")
	})
	a.Attribute("url", d.String, "The URL of the codebase ", func() {
		a.Example("git@github.com:fabric8-services/fabric8-wit.git")
	})
	a.Attribute("stackId", d.String, "The stack id of the codebase ", func() {
		a.Example("java-centos")
	})
	a.Attribute("createdAt", d.DateTime, "When the codebase was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("last_used_workspace", d.String, "The last used workspace name of the codebase ", func() {
		a.Example("java-centos")
	})
})

var codebaseLinks = a.Type("CodebaseLinks", func() {
	a.UseTrait("GenericLinksTrait")
	a.Attribute("edit", d.String)
})
var codebaseRelationships = a.Type("CodebaseRelations", func() {
	a.Attribute("space", relationGeneric, "This defines the owning space")
})

var codebaseListMeta = a.Type("CodebaseListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var workspace = a.Type("Workspace", func() {
	a.Description(`JSONAPI store for the data of a workspace.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("workspaces")
	})
	a.Attribute("attributes", workspaceAttributes)
	a.Attribute("links", workspaceLinks)
	a.Required("type", "attributes")
})

var workspaceAttributes = a.Type("WorkspaceAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a workspace. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The workspace name", func() {
		a.Example("test")
	})
	a.Attribute("description", d.String, "The URL of the codebase ", func() {
		a.Example("")
	})
})

var workspaceLinks = a.Type("WorkspaceLinks", func() {
	a.Attribute("open", d.String)
})

var workspaceEditLinks = a.Type("WorkspaceEditLinks", func() {
	a.Attribute("create", d.String)
})

var workspaceOpenLinks = a.Type("WorkspaceOpenLinks", func() {
	a.Attribute("open", d.String)
})

var codebaseList = JSONList(
	"Codebase", "Holds the list of codebases",
	codebase,
	pagingLinks,
	codebaseListMeta)

var codebaseSingle = JSONSingle(
	"Codebase", "Holds a single codebase",
	codebase,
	nil)

var workspaceList = JSONList(
	"Workspace", "Holds the list of workspaces related to a codebase",
	workspace,
	workspaceEditLinks,
	nil)

var workspaceOpen = a.MediaType("application/vnd.workspaceopen+json", func() {
	a.TypeName("WorkspaceOpen")
	a.Description(`JSONAPI store for the links of a workspace.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("links", workspaceOpenLinks)
	a.View("default", func() {
		a.Attribute("links")
	})
})

// new version of "list" for migration
var _ = a.Resource("codebase", func() {
	a.BasePath("/codebases")
	a.Action("show", func() {
		a.Routing(
			a.GET("/:codebaseID"),
		)
		a.Description("Retrieve codebase with given id.")
		a.Params(func() {
			a.Param("codebaseID", d.UUID, "Codebase Identifier")
		})
		a.Response(d.OK, func() {
			a.Media(codebaseSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("edit", func() {
		a.Security("jwt")
		a.Routing(
			a.GET("/:codebaseID/edit"),
		)
		a.Description("Trigger edit of a given codebase.")
		a.Params(func() {
			a.Param("codebaseID", d.UUID, "Codebase Identifier")
		})
		a.Response(d.OK, func() {
			a.Media(workspaceList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:codebaseID/create"),
		)
		a.Description("Trigger create a worksapce for a codebase.")
		a.Params(func() {
			a.Param("codebaseID", d.UUID, "Codebase Identifier")
		})
		a.Response(d.OK, func() {
			a.Media(workspaceOpen)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
	a.Action("open", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("/:codebaseID/open/:workspaceID"),
		)
		a.Description("Trigger open of a given worksapce for a codebase.")
		a.Params(func() {
			a.Param("codebaseID", d.UUID, "Codebase Identifier")
			a.Param("workspaceID", d.String, "Workspace Identifier")
		})
		a.Response(d.OK, func() {
			a.Media(workspaceOpen)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})

// new version of "list" for migration
var _ = a.Resource("space_codebases", func() {
	a.Parent("space")

	a.Action("list", func() {
		a.Routing(
			a.GET("codebases"),
		)
		a.Description("List codebases.")
		a.Params(func() {
			a.Param("page[offset]", d.String, "Paging start position")
			a.Param("page[limit]", d.Integer, "Paging size")
		})
		a.Response(d.OK, func() {
			a.Media(codebaseList)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("codebases"),
		)
		a.Description("Create codebase.")
		a.Payload(codebaseSingle)
		a.Response(d.Created, "/codebases/.*", func() {
			a.Media(codebaseSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
})
