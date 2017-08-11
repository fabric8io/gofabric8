package controller

import (
	"context"
	"fmt"
	"html"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// CommentsController implements the comments resource.
type CommentsController struct {
	*goa.Controller
	db           application.DB
	notification notification.Channel
	config       CommentsControllerConfiguration
}

// CommentsControllerConfiguration the configuration for CommentsController
type CommentsControllerConfiguration interface {
	GetCacheControlComments() string
}

// NewCommentsController creates a comments controller.
func NewCommentsController(service *goa.Service, db application.DB, config CommentsControllerConfiguration) *CommentsController {
	return NewNotifyingCommentsController(service, db, &notification.DevNullChannel{}, config)
}

// NewNotifyingCommentsController creates a comments controller with notification broadcast.
func NewNotifyingCommentsController(service *goa.Service, db application.DB, notificationChannel notification.Channel, config CommentsControllerConfiguration) *CommentsController {
	n := notificationChannel
	if n == nil {
		n = &notification.DevNullChannel{}
	}
	return &CommentsController{
		Controller:   service.NewController("CommentsController"),
		db:           db,
		notification: n,
		config:       config,
	}
}

// Show runs the show action.
func (c *CommentsController) Show(ctx *app.ShowCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		cmt, err := appl.Comments().Load(ctx, ctx.CommentID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}
		return ctx.ConditionalRequest(*cmt, c.config.GetCacheControlComments, func() error {
			res := &app.CommentSingle{}
			// This code should change if others type of parents than WI are allowed
			includeParentWorkItem, err := CommentIncludeParentWorkItem(ctx, appl, cmt)
			if err != nil {
				return errors.NewNotFoundError("comment parentID", cmt.ParentID.String())
			}
			res.Data = ConvertComment(
				ctx.RequestData,
				*cmt,
				includeParentWorkItem)
			return ctx.OK(res)
		})
	})
}

// Update does PATCH comment
func (c *CommentsController) Update(ctx *app.UpdateCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var cm *comment.Comment
	var wi *workitem.WorkItem
	var editorIsCreator bool
	// Following transaction verifies if a user is allowed to update or not
	err = application.Transactional(c.db, func(appl application.Application) error {
		cm, err = appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		if *identityID == cm.Creator {
			editorIsCreator = true
			return nil
		}
		wi, err = appl.WorkItems().LoadByID(ctx.Context, cm.ParentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// User is allowed to update if user is creator of the comment OR user is a space collaborator
	if !editorIsCreator {
		authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
		}
		if !authorized {
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not a space collaborator"))
		}
	}
	result := c.performUpdate(ctx, cm, identityID)
	if ctx.ResponseData.Status == 200 {
		c.notification.Send(ctx, notification.NewCommentUpdated(cm.ID.String()))
	}
	return result
}

func (c *CommentsController) performUpdate(ctx *app.UpdateCommentsContext, cm *comment.Comment, identityID *uuid.UUID) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		cm.Body = *ctx.Payload.Data.Attributes.Body
		cm.Markup = rendering.NilSafeGetMarkup(ctx.Payload.Data.Attributes.Markup)
		err := appl.Comments().Save(ctx.Context, cm, *identityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		// This code should change if others type of parents than WI are allowed
		includeParentWorkItem, err := CommentIncludeParentWorkItem(ctx, appl, cm)
		if err != nil {
			return errors.NewNotFoundError("comment parentID", cm.ParentID.String())
		}

		res := &app.CommentSingle{
			Data: ConvertComment(ctx.RequestData, *cm, includeParentWorkItem),
		}
		return ctx.OK(res)
	})
}

// Delete does DELETE comment
func (c *CommentsController) Delete(ctx *app.DeleteCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	var cm *comment.Comment
	var wi *workitem.WorkItem
	var userIsCreator bool
	// Following transaction verifies if a user is allowed to delete or not
	err = application.Transactional(c.db, func(appl application.Application) error {
		cm, err = appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		if *identityID == cm.Creator {
			userIsCreator = true
			return nil
		}
		wi, err = appl.WorkItems().LoadByID(ctx.Context, cm.ParentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// User is allowed to delete if user is creator of the comment OR user is a space collaborator
	if userIsCreator {
		return c.performDelete(ctx, cm, identityID)
	}

	authorized, err := authz.Authorize(ctx, wi.SpaceID.String())
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	if !authorized {
		return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not a space collaborator"))
	}
	return c.performDelete(ctx, cm, identityID)
}

func (c *CommentsController) performDelete(ctx *app.DeleteCommentsContext, cm *comment.Comment, identityID *uuid.UUID) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Comments().Delete(ctx.Context, cm.ID, *identityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK([]byte{})
	})
}

// CommentConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
type CommentConvertFunc func(*goa.RequestData, *comment.Comment, *app.Comment)

// ConvertComments converts between internal and external REST representation
func ConvertComments(request *goa.RequestData, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertComment(request, c, additional...))
	}
	return cs
}

// ConvertCommentsResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentsResourceID(request *goa.RequestData, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertCommentResourceID(request, c, additional...))
	}
	return cs
}

// ConvertCommentResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentResourceID(request *goa.RequestData, comment comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
	}
	for _, add := range additional {
		add(request, &comment, c)
	}
	return c
}

// ConvertComment converts between internal and external REST representation
func ConvertComment(request *goa.RequestData, comment comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	userType := APIStringTypeUser
	creatorID := comment.Creator.String()
	relatedURL := rest.AbsoluteURL(request, app.CommentsHref(comment.ID))
	markup := rendering.NilSafeGetMarkup(&comment.Markup)
	bodyRendered := rendering.RenderMarkupToHTML(html.EscapeString(comment.Body), comment.Markup)
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, creatorID))
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
		Attributes: &app.CommentAttributes{
			Body:         &comment.Body,
			BodyRendered: &bodyRendered,
			Markup:       &markup,
			CreatedAt:    &comment.CreatedAt,
			UpdatedAt:    &comment.UpdatedAt,
		},
		Relationships: &app.CommentRelations{
			Creator: &app.RelationGeneric{
				Data: &app.GenericData{
					Type: &userType,
					ID:   &creatorID,
					Links: &app.GenericLinks{
						Related: &relatedCreatorLink,
					},
				},
			},
			CreatedBy: &app.CommentCreatedBy{ // Keep old API style until all cients are updated
				Data: &app.IdentityRelationData{
					Type: userType,
					ID:   &comment.Creator,
				},
				Links: &app.GenericLinks{
					Related: &relatedCreatorLink,
				},
			},
		},
		Links: &app.GenericLinks{
			Self:    &relatedURL,
			Related: &relatedURL,
		},
	}
	for _, add := range additional {
		add(request, &comment, c)
	}
	return c
}

// HrefFunc generic function to greate a relative Href to a resource
type HrefFunc func(id interface{}) string

// CommentIncludeParentWorkItem includes a "parent" relation to a WorkItem
func CommentIncludeParentWorkItem(ctx context.Context, appl application.Application, c *comment.Comment) (CommentConvertFunc, error) {
	return func(request *goa.RequestData, comment *comment.Comment, data *app.Comment) {
		hrefFunc := func(obj interface{}) string {
			return fmt.Sprintf(app.WorkitemHref("%v"), obj)
		}
		CommentIncludeParent(request, comment, data, hrefFunc, APIStringTypeWorkItem)
	}, nil
}

// CommentIncludeParent adds the "parent" relationship to this Comment
func CommentIncludeParent(request *goa.RequestData, comment *comment.Comment, data *app.Comment, ref HrefFunc, parentType string) {
	parentSelf := rest.AbsoluteURL(request, ref(comment.ParentID))
	parentID := comment.ParentID.String()
	data.Relationships.Parent = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &parentType,
			ID:   &parentID,
		},
		Links: &app.GenericLinks{
			Self: &parentSelf,
		},
	}
}
