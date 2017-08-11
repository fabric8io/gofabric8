package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var auth = a.Type("Auth", func() {
	a.Description(`REST API for accessing Tokens using a REST API like KeyCloak`)
	a.Attribute("type", d.String, func() {
		a.Enum("auth")
	})
	a.Attribute("id", d.UUID, "ID of auth", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", authAttributes)
	a.Attribute("links", genericLinks)
	a.Required("type", "attributes")
})

var authAttributes = a.Type("AuthAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a Auth. See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("email", d.String, "The auth name", func() {
		a.Example("Email for the auth")
	})
	a.Attribute("created-at", d.DateTime, "When the auth was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
})

var _ = a.Resource("auth", func() {
	a.BasePath("/auth/realms/:realm/broker/:broker/token")
	a.Action("authToken", func() {
		a.Security("jwt")
		a.Routing(
			a.GET(""),
		)

		a.Description("Get the token for the given realm and broker")
		a.Response(d.Accepted)
		a.Response(d.Conflict)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
	})
})
