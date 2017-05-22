package design

import (
	"strings"

	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

//#############################################################################
//
// 			JSONAPI common
//
//#############################################################################

// JSONAPILink represents a JSONAPI link object (see http://jsonapi.org/format/#document-links)
var JSONAPILink = a.Type("JSONAPILink", func() {
	a.Description(`See also http://jsonapi.org/format/#document-links.`)
	a.Attribute("href", d.String, "a string containing the link's URL.", func() {
		a.Example("http://example.com/articles/1/comments")
	})
	a.Attribute("meta", a.HashOf(d.String, d.Any), "a meta object containing non-standard meta-information about the link.")
})

// JSONAPIError represents a JSONAPI error object (see http://jsonapi.org/format/#error-objects)
var JSONAPIError = a.Type("JSONAPIError", func() {
	a.Description(`Error objects provide additional information about problems encountered while
performing an operation. Error objects MUST be returned as an array keyed by errors in the
top level of a JSON API document.

See. also http://jsonapi.org/format/#error-objects.`)

	a.Attribute("id", d.String, "a unique identifier for this particular occurrence of the problem.")
	a.Attribute("links", a.HashOf(d.String, JSONAPILink), `a links object containing the following members:
* about: a link that leads to further details about this particular occurrence of the problem.`)
	a.Attribute("status", d.String, "the HTTP status code applicable to this problem, expressed as a string value.")
	a.Attribute("code", d.String, "an application-specific error code, expressed as a string value.")
	a.Attribute("title", d.String, `a short, human-readable summary of the problem that SHOULD NOT
change from occurrence to occurrence of the problem, except for purposes of localization.`)
	a.Attribute("detail", d.String, `a human-readable explanation specific to this occurrence of the problem.
Like title, this field’s value can be localized.`)
	a.Attribute("source", a.HashOf(d.String, d.Any), `an object containing references to the source of the error,
optionally including any of the following members

* pointer: a JSON Pointer [RFC6901] to the associated entity in the request document [e.g. "/data" for a primary data object,
           or "/data/attributes/title" for a specific attribute].
* parameter: a string indicating which URI query parameter caused the error.`)
	a.Attribute("meta", a.HashOf(d.String, d.Any), "a meta object containing non-standard meta-information about the error")

	a.Required("detail")
})

// JSONAPIErrors is an array of JSONAPI error objects
var JSONAPIErrors = a.MediaType("application/vnd.jsonapierrors+json", func() {
	a.UseTrait("jsonapi-media-type")
	a.TypeName("JSONAPIErrors")
	a.Description(``)
	a.Attributes(func() {
		a.Attribute("errors", a.ArrayOf(JSONAPIError))
		a.Required("errors")
	})
	a.View("default", func() {
		a.Attribute("errors")
		a.Required("errors")
	})
})

// relationGeneric is a top level structure for 'other' relationships
var relationGeneric = a.Type("RelationGeneric", func() {
	a.Attribute("data", genericData)
	a.Attribute("links", genericLinks)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
})

// relationGeneric is a top level structure for 'other' relationships
var relationGenericList = a.Type("RelationGenericList", func() {
	a.Attribute("data", a.ArrayOf(genericData))
	a.Attribute("links", genericLinks)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
})

// genericData defines what is needed inside Generic Relationship
var genericData = a.Type("GenericData", func() {
	a.Attribute("type", d.String)
	a.Attribute("id", d.String, "UUID of the object", func() {
		a.Example("6c5610be-30b2-4880-9fec-81e4f8e4fd76")
	})
	a.Attribute("links", genericLinks)
})

// genericLinks defines generic relations links
var genericLinks = a.Type("GenericLinks", func() {
	a.Attribute("self", d.String)
	a.Attribute("related", d.String)
	a.Attribute("meta", a.HashOf(d.String, d.Any))
})

// JSONResourceObject creates a single resource object
func JSONResourceObject(name string, attributes *d.UserTypeDefinition, relationships *d.UserTypeDefinition) *d.UserTypeDefinition {
	return a.Type(name, func() {
		a.Attribute("type", d.String, func() {
			a.Enum(strings.ToLower(name) + "s")
		})
		a.Attribute("id", d.String, "ID of the "+name, func() {
			a.Example("42")
		})
		a.Attribute("attributes", attributes)
		if relationships != nil {
			a.Attribute("relationships", relationships)
		}
		a.Required("type", "id", "attributes")
	})
}

// JSONList creates a UserTypeDefinition
func JSONList(name, description string, data *d.UserTypeDefinition, links *d.UserTypeDefinition, meta *d.UserTypeDefinition) *d.MediaTypeDefinition {
	return a.MediaType("application/vnd."+strings.ToLower(name)+"list+json", func() {
		a.UseTrait("jsonapi-media-type")
		a.TypeName(name + "List")
		a.Description(description)
		if links != nil {
			a.Attribute("links", links)
		}
		if meta != nil {
			a.Attribute("meta", meta)
		}
		a.Attribute("data", a.ArrayOf(data))
		a.Attribute("included", a.ArrayOf(d.Any), "An array of mixed types")
		a.Required("data")

		a.View("default", func() {
			if links != nil {
				a.Attribute("links")
			}
			if meta != nil {
				a.Attribute("meta")
			}
			a.Attribute("data")
			a.Attribute("included")
			a.Required("data")
		})
	})

}

// JSONSingle creates a Single
func JSONSingle(name, description string, data *d.UserTypeDefinition, links *d.UserTypeDefinition) *d.MediaTypeDefinition {
	// WorkItemSingle is the media type for work items
	return a.MediaType("application/vnd."+strings.ToLower(name)+"+json", func() {
		a.UseTrait("jsonapi-media-type")
		a.TypeName(name + "Single")
		a.Description(description)
		if links != nil {
			a.Attribute("links", links)
		}
		a.Attribute("data", data)
		a.Attribute("included", a.ArrayOf(d.Any), "An array of mixed types")
		a.Required("data")
		a.View("default", func() {
			if links != nil {
				a.Attribute("links")
			}
			a.Attribute("data")
			a.Attribute("included")
			a.Required("data")
		})
	})
}
