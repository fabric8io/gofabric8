package design

import (
	d "github.com/goadesign/goa/design"
	a "github.com/goadesign/goa/design/apidsl"
)

var comment = a.Type("Comment", func() {
	a.Description(`JSONAPI store for the data of a comment.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("comments")
	})
	a.Attribute("id", d.UUID, "ID of comment", func() {
		a.Example("40bbdd3d-8b5d-4fd6-ac90-7236b669af04")
	})
	a.Attribute("attributes", commentAttributes)
	a.Attribute("relationships", commentRelationships)
	a.Attribute("links", genericLinks)
	a.Required("type")
})

var createComment = a.Type("CreateComment", func() {
	a.Description(`JSONAPI store for the data of a comment.  See also http://jsonapi.org/format/#document-resource-object`)
	a.Attribute("type", d.String, func() {
		a.Enum("comments")
	})
	a.Attribute("attributes", createCommentAttributes)
	a.Required("type", "attributes")
})

var commentAttributes = a.Type("CommentAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" of a comment. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("created-at", d.DateTime, "When the comment was created", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("updated-at", d.DateTime, "When the comment was updated", func() {
		a.Example("2016-11-29T23:18:14Z")
	})
	a.Attribute("body", d.String, "The comment body", func() {
		a.Example("This is really interesting")
	})
	a.Attribute("body.rendered", d.String, "The comment body rendered in HTML", func() {
		a.Example("<p>This is really interesting</p>\n")
	})
	a.Attribute("markup", d.String, "The comment markup associated with the body", func() {
		a.Example("Markdown")
	})
})

var createCommentAttributes = a.Type("CreateCommentAttributes", func() {
	a.Description(`JSONAPI store for all the "attributes" for creating a comment. +See also see http://jsonapi.org/format/#document-resource-object-attributes`)
	a.Attribute("body", d.String, "The comment body", func() {
		a.MinLength(1) // Empty comment not allowed
		a.Example("This is really interesting")
	})
	a.Attribute("markup", d.String, "The comment markup associated with the body", func() {
		a.Example("Markdown")
	})
	a.Required("body")
})

var commentRelationships = a.Type("CommentRelations", func() {
	a.Attribute("creator", relationGeneric, "This defines the creator of the comment")
	a.Attribute("created-by", commentCreatedBy, "DEPRECATED. This defines the creator of the comment.")
	a.Attribute("parent", relationGeneric, "This defines the owning resource of the comment")
})

var commentCreatedBy = a.Type("CommentCreatedBy", func() {
	a.Attribute("data", identityRelationData)
	a.Required("data")
	a.Attribute("links", genericLinks)
})

var identityRelationData = a.Type("IdentityRelationData", func() {
	a.Attribute("id", d.UUID, "unique id for the user identity")
	a.Attribute("type", d.String, "type of the user identity", func() {
		a.Enum("identities")
	})
	a.Required("type")
})

var commentRelationshipsArray = JSONList(
	"CommentRelationship", "Holds the response of comments",
	comment,
	genericLinks,
	commentListMeta,
)

var commentListMeta = a.Type("CommentListMeta", func() {
	a.Attribute("totalCount", d.Integer)
	a.Required("totalCount")
})

var commentArray = JSONList(
	"Comment", "Holds the response of comments",
	comment,
	pagingLinks,
	commentListMeta,
)

var commentSingle = JSONSingle(
	"Comment", "Holds the response of a single comment",
	comment,
	nil,
)
var createSingleComment = JSONSingle(
	"CreateSingle", "Holds the create data for a comment",
	createComment,
	nil,
)

var _ = a.Resource("comments", func() {
	a.BasePath("/comments")

	a.Action("show", func() {
		a.Routing(
			a.GET("/:commentId"),
		)
		a.Params(func() {
			a.Param("commentId", d.UUID, "commentId")
		})
		a.Description("Retrieve comment with given commentId.")
		a.UseTrait("conditional")
		a.Response(d.OK, commentSingle)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("update", func() {
		a.Security("jwt")
		a.Routing(
			a.PATCH("/:commentId"),
		)
		a.Description("update the comment with the given commentId.")
		a.Params(func() {
			a.Param("commentId", d.UUID, "commentId")
		})
		a.Payload(commentSingle)
		a.Response(d.OK, func() {
			a.Media(commentSingle)
		})
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})
	a.Action("delete", func() {
		a.Security("jwt")
		a.Routing(
			a.DELETE("/:commentId"),
		)
		a.Description("Delete work item with given id.")
		a.Params(func() {
			a.Param("commentId", d.UUID, "commentId")
		})
		a.Response(d.OK)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.Forbidden, JSONAPIErrors)
	})

})

var _ = a.Resource("work_item_comments", func() {
	a.Parent("workitem")

	a.Action("list", func() {
		a.Routing(
			a.GET("comments"),
		)
		a.Description("List comments associated with the given work item")
		a.Params(func() {
			a.Param("page[offset]", d.String, `Paging start position is a string pointing to
			the beginning of pagination.  The value starts from 0 onwards.`)
			a.Param("page[limit]", d.Integer, `Paging size is the number of items in a page`)
		})
		a.UseTrait("conditional")
		a.Response(d.OK, commentArray)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
	a.Action("relations", func() {
		a.Routing(
			a.GET("relationships/comments"),
		)
		a.Description("List comments associated with the given work item")
		a.Params(func() {
			a.Param("page[offset]", d.String, `Paging start position is a string pointing to
				the beginning of pagination.  The value starts from 0 onwards.`)
			a.Param("page[limit]", d.Integer, `Paging size is the number of items in a page`)
		})
		a.UseTrait("conditional")
		a.Response(d.OK, commentRelationshipsArray)
		a.Response(d.NotModified)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})

	a.Action("create", func() {
		a.Security("jwt")
		a.Routing(
			a.POST("comments"),
		)
		a.Description("Creates a comment associated with the given work item")
		a.Response(d.OK, func() {
			a.Media(commentSingle)
		})
		a.Payload(createSingleComment)
		a.Response(d.BadRequest, JSONAPIErrors)
		a.Response(d.Unauthorized, JSONAPIErrors)
		a.Response(d.InternalServerError, JSONAPIErrors)
		a.Response(d.NotFound, JSONAPIErrors)
	})
})
