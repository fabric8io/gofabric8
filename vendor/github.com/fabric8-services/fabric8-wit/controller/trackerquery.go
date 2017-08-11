package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

type trackerQueryConfiguration interface {
	GetGithubAuthToken() string
}

// TrackerqueryController implements the trackerquery resource.
type TrackerqueryController struct {
	*goa.Controller
	db            application.DB
	scheduler     *remoteworkitem.Scheduler
	configuration trackerQueryConfiguration
}

func getAccessTokensForTrackerQuery(configuration trackerQueryConfiguration) map[string]string {
	tokens := map[string]string{
		remoteworkitem.ProviderGithub: configuration.GetGithubAuthToken(),
		// add tokens for other types
	}
	return tokens
}

// NewTrackerqueryController creates a trackerquery controller.
func NewTrackerqueryController(service *goa.Service, db application.DB, scheduler *remoteworkitem.Scheduler, configuration trackerQueryConfiguration) *TrackerqueryController {
	return &TrackerqueryController{Controller: service.NewController("TrackerqueryController"), db: db, scheduler: scheduler, configuration: configuration}
}

// Create runs the create action.
func (c *TrackerqueryController) Create(ctx *app.CreateTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		tq, err := appl.TrackerQueries().Create(ctx.Context, ctx.Payload.Query, ctx.Payload.Schedule, ctx.Payload.TrackerID, *ctx.Payload.Relationships.Space.Data.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		ctx.ResponseData.Header().Set("Location", app.TrackerqueryHref(tq.ID))
		return ctx.Created(tq)
	})
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return result
}

// Show runs the show action.
func (c *TrackerqueryController) Show(ctx *app.ShowTrackerqueryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		tq, err := appl.TrackerQueries().Load(ctx.Context, ctx.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				log.Error(ctx, map[string]interface{}{
					"tracker_id": ctx.ID,
				}, "tracker query controller not found")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				return errs.WithStack(err)
			}
		}
		return ctx.OK(tq)
	})
}

// Update runs the update action.
func (c *TrackerqueryController) Update(ctx *app.UpdateTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {

		toSave := app.TrackerQuery{
			ID:            ctx.ID,
			Query:         ctx.Payload.Query,
			Schedule:      ctx.Payload.Schedule,
			TrackerID:     ctx.Payload.TrackerID,
			Relationships: ctx.Payload.Relationships,
		}
		tq, err := appl.TrackerQueries().Save(ctx.Context, toSave)

		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.BadParameterError, remoteworkitem.ConversionError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest(err.Error()))
				return ctx.BadRequest(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		return ctx.OK(tq)
	})
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return result
}

// Delete runs the delete action.
func (c *TrackerqueryController) Delete(ctx *app.DeleteTrackerqueryContext) error {
	result := application.Transactional(c.db, func(appl application.Application) error {
		err := appl.TrackerQueries().Delete(ctx.Context, ctx.ID)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case remoteworkitem.NotFoundError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrNotFound(err.Error()))
				return ctx.NotFound(jerrors)
			default:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}
		return ctx.OK([]byte{})
	})
	accessTokens := getAccessTokensForTrackerQuery(c.configuration) //configuration.GetGithubAuthToken()
	c.scheduler.ScheduleAllQueries(ctx, accessTokens)
	return result
}

// List runs the list action.
func (c *TrackerqueryController) List(ctx *app.ListTrackerqueryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		result, err := appl.TrackerQueries().List(ctx.Context)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(fmt.Sprintf("Error listing tracker queries: %s", err.Error())))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(result)
	})

}
