package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.API("wit", func() {
	a.Title("Fabric8-wit: One to rule them all")
	a.Description("The next big thing")
	a.Version("1.0")
	a.Host("openshift.io")
	a.Scheme("http")
	a.BasePath("/api")
	a.Consumes("application/json")
	a.Produces("application/json")

	a.License(func() {
		a.Name("Apache License Version 2.0")
		a.URL("http://www.apache.org/licenses/LICENSE-2.0")
	})
	a.Origin("/[.*openshift.io|localhost]/", func() {
		a.Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		a.Headers("X-Request-Id", "Content-Type", "Authorization", "If-None-Match", "If-Modified-Since")
		a.MaxAge(600)
		a.Credentials()
	})

	a.Trait("GenericLinksTrait", func() {
		a.Attribute("self", d.String)
		a.Attribute("related", d.String)
		a.Attribute("meta", a.HashOf(d.String, d.Any))
	})

	a.Trait("jsonapi-media-type", func() {
		a.ContentType("application/vnd.api+json")
	})

	a.Trait("conditional", func() {
		a.Headers(func() {
			a.Header("If-Modified-Since", d.String)
			a.Header("If-None-Match", d.String)
		})
	})

	a.JWTSecurity("jwt", func() {
		a.Description("JWT Token Auth")
		a.TokenURL("/api/login/authorize")
		a.Header("Authorization")
	})

	a.ResponseTemplate(d.OK, func() {
		a.Description("Resource created")
		a.Status(200)
		a.Headers(func() {
			a.Header("Last-Modified", d.DateTime)
			a.Header("ETag")
			a.Header("Cache-Control")
		})
	})

	a.ResponseTemplate(d.Created, func(pattern string) {
		a.Description("Resource created")
		a.Status(201)
		a.Headers(func() {
			a.Header("Location", d.String, "href to created resource", func() {
				a.Pattern(pattern)
			})
		})
	})
})
