package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var _ = a.Resource("userspace", func() {
	a.BasePath("/userspace")

	a.Action("create", func() {
		a.Routing(
			a.PUT("/*"),
		)
		a.Description("Data dump endpoint ")
		a.Payload(a.HashOf(d.String, d.Any))
		a.Response(d.NoContent)
		a.Response(d.InternalServerError)
	})
	a.Action("show", func() {
		a.Routing(
			a.GET("/*"),
		)
		a.Description("Data dump endpoint ")
		a.Response(d.OK, a.HashOf(d.String, d.Any))
		a.Response(d.InternalServerError)
		a.Response(d.NotFound)
	})
})
