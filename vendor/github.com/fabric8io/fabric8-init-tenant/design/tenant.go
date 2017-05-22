package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var tenant = a.Type("Tenant", func() {
	a.Description(`JSONAPI for the tenant object. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("tenant")
	})
	a.Attribute("id", d.UUID, "ID of tenant", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", tenantAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var tenantAttributes = a.Type("TenantAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Tenant. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("email", d.String, "The tenant name", func() {
		a.Example("Email for the tenant")
	})
	a.Attribute("created-at", d.DateTime, "When the tenant was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("namespaces", a.ArrayOf(namespaceAttributes), "The tenant namespaces", func() {
	})
})

var namespaceAttributes = a.Type("NamespaceAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Tenant namespace. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("name", d.String, "The namespace name", func() {
		a.Example("Name for the tenant namespace")
	})
	a.Attribute("created-at", d.DateTime, "When the tenant was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the tenant was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("version", d.String, "The namespaces version", func() {
	})
	a.Attribute("state", d.String, "The namespaces state", func() {
	})
	a.Attribute("cluster-url", d.String, "The cluster url", func() {
	})
	a.Attribute("type", d.String, "The tenant namespaces", func() {
		a.Enum("che", "jenkins", "stage", "test", "run")
	})
})

var tenantSingle = JSONSingle(
	"tenant", "Holds a single Tenant",
	tenant,
	nil)

var _ = a.Resource("tenant", func() {
	a.BasePath("/tenant")
	a.Action("setup", func() {
		a.Security("jwt")
		a.Routing(
			a.POST(""),
		)

		a.Description("Initialize new tenant environment.")
		a.Response(d.Accepted)
		a.Response(d.Conflict)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH(""),
		)

		a.Description("Initialize new tenant environment.")
		a.Response(d.Accepted)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})

	a.Action("show", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)

		a.Description("Initialize new tenant environment.")
		a.Response(d.OK, tenantSingle)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
