package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/goadesign/goa"
)

// SpaceCodebasesController implements the space-codebases resource.
type SpaceCodebasesController struct {
	*goa.Controller
	db application.DB
}

// NewSpaceCodebasesController creates a space-codebases controller.
func NewSpaceCodebasesController(service *goa.Service, db application.DB) *SpaceCodebasesController {
	return &SpaceCodebasesController{Controller: service.NewController("SpaceCodebasesController"), db: db}
}

// Create runs the create action.
func (c *SpaceCodebasesController) Create(ctx *app.CreateSpaceCodebasesContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	// Validate Request
	if ctx.Payload.Data == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data", nil).Expected("not nil"))
	}
	reqIter := ctx.Payload.Data
	if reqIter.Attributes.Type == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.type", nil).Expected("not nil"))
	}
	if reqIter.Attributes.URL == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("data.attributes.url", nil).Expected("not nil"))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		space, err := appl.Spaces().Load(ctx, ctx.SpaceID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		if *identityID != space.OwnerId {
			return jsonapi.JSONErrorResponse(ctx, errors.NewForbiddenError("user is not the space owner"))
		}

		newCodeBase := codebase.Codebase{
			SpaceID: ctx.SpaceID,
			Type:    *reqIter.Attributes.Type,
			URL:     *reqIter.Attributes.URL,
			StackID: reqIter.Attributes.StackID,
		}
		err = appl.Codebases().Create(ctx, &newCodeBase)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.CodebaseSingle{
			Data: ConvertCodebase(ctx.RequestData, &newCodeBase),
		}
		ctx.ResponseData.Header().Set("Location", rest.AbsoluteURL(ctx.RequestData, app.CodebaseHref(res.Data.ID)))
		return ctx.Created(res)
	})
}

// List runs the list action.
func (c *SpaceCodebasesController) List(ctx *app.ListSpaceCodebasesContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.Spaces().CheckExists(ctx, ctx.SpaceID.String())
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		res := &app.CodebaseList{}
		res.Data = []*app.Codebase{}

		codebases, tc, err := appl.Codebases().List(ctx, ctx.SpaceID, &offset, &limit)
		count := int(tc)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		res.Meta = &app.CodebaseListMeta{TotalCount: count}
		res.Data = ConvertCodebases(ctx.RequestData, codebases)
		res.Links = &app.PagingLinks{}
		setPagingLinks(res.Links, buildAbsoluteURL(ctx.RequestData), len(codebases), offset, limit, count)

		return ctx.OK(res)
	})
}
