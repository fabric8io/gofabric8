package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var tenantStatus = a.Type("TenantStatus", func() {
	a.Description(`JSONAPI for the tenantStatus object. See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("tenantStatus")
	})
	a.Attribute("id", d.UUID, "ID of tenantStatus", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669afff")
	})
	a.Attribute("attributes", tenantStatusAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var tenantStatusAttributes = a.Type("TenantStatusAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a TenantStatus. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("message", d.String, "The status message", func() {
		a.Example("Status of the tenant connection")
	})
	a.Attribute("error", d.String, "The error message", func() {
		a.Example("Error message for tenant connection")
	})
})

var tenantStatusSingle = JSONSingle(
	"tenantStatus", "Holds a single TenantStatus",
	tenantStatus,
	nil)

var _ = a.Resource("tenantKube", func() {
	a.BasePath("/api/tenant/kubeconnect")
	a.Action("kubeConnected", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)

		a.Description("Checks if the kubernetes tenant is connected with KeyCloak.")
		a.Response(d.Accepted)
		a.Response(d.OK, tenantStatusSingle)
		a.Response(d.Conflict, tenantStatusSingle)
		a.Response(d.InternalServerError, JSONAPIErrors)
	})
})
