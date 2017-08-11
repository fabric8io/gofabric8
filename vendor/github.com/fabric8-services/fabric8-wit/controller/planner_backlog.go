package controller

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	query "github.com/fabric8-services/fabric8-wit/query/simple"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// PlannerBacklogController implements the planner_backlog resource.
type PlannerBacklogController struct {
	*goa.Controller
	db     application.DB
	config PlannerBacklogControllerConfig
}

type PlannerBacklogControllerConfig interface {
	GetCacheControlWorkItems() string
}

// NewPlannerBacklogController creates a planner_backlog controller.
func NewPlannerBacklogController(service *goa.Service, db application.DB, config PlannerBacklogControllerConfig) *PlannerBacklogController {
	return &PlannerBacklogController{
		Controller: service.NewController("PlannerBacklogController"),
		db:         db,
		config:     config,
	}
}

func (c *PlannerBacklogController) List(ctx *app.ListPlannerBacklogContext) error {
	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)
	exp, err := query.Parse(ctx.Filter)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("could not parse filter", err))
	}
	if ctx.FilterAssignee != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*ctx.FilterAssignee})))
	}
	if ctx.FilterWorkitemtype != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*ctx.FilterWorkitemtype})))
	}
	if ctx.FilterArea != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*ctx.FilterArea))))
	}

	// Get the list of work items for the following criteria
	result, count, err := getBacklogItems(ctx.Context, c.db, ctx.SpaceID, exp, &offset, &limit)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}
	return ctx.ConditionalEntities(result, c.config.GetCacheControlWorkItems, func() error {
		response := app.WorkItemList{
			Data:  ConvertWorkItems(ctx.RequestData, result),
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), count, offset, limit, count)
		return ctx.OK(&response)
	})

}

// generateBacklogExpression creates the expression to query for backlog items
func generateBacklogExpression(ctx context.Context, db application.DB, spaceID uuid.UUID, exp criteria.Expression) (criteria.Expression, error) {
	if exp != nil {
		exp = criteria.And(exp, criteria.Not(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateClosed)))
	} else {
		exp = criteria.Not(criteria.Field(workitem.SystemState), criteria.Literal(workitem.SystemStateClosed))
	}

	err := application.Transactional(db, func(appl application.Application) error {
		// Get the root iteration
		iteration, err := appl.Iterations().Root(ctx, spaceID)
		if err != nil {
			return errs.Wrap(err, "unable to fetch root iteration")
		}
		exp = criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(iteration.ID.String()))

		// Get the list of work item types that derive of PlannerItem in the space
		var expWits criteria.Expression
		wits, err := appl.WorkItemTypes().ListPlannerItems(ctx, spaceID)
		if err != nil {
			return errs.Wrap(err, "unable to fetch work item types that derive from planner item")
		}
		if len(wits) >= 1 {
			expWits = criteria.Equals(criteria.Field("Type"), criteria.Literal(wits[0].ID.String()))
			for _, wit := range wits[1:] {
				witIDStr := wit.ID.String()
				expWits = criteria.Or(expWits, criteria.Equals(criteria.Field("Type"), criteria.Literal(witIDStr)))
			}
			exp = criteria.And(exp, expWits)
		}
		if len(wits) == 0 {
			// We set exp to nil to return an empty array
			exp = nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return exp, nil
}

func getBacklogItems(ctx context.Context, db application.DB, spaceID uuid.UUID, exp criteria.Expression, offset *int, limit *int) ([]workitem.WorkItem, int, error) {
	result := []workitem.WorkItem{}
	count := 0

	backlogExp, err := generateBacklogExpression(ctx, db, spaceID, exp)
	if err != nil || backlogExp == nil {
		return result, count, err
	}

	err = application.Transactional(db, func(appl application.Application) error {
		// Get the list of work items for the following criteria
		result, count, err = appl.WorkItems().List(ctx, spaceID, backlogExp, nil, offset, limit)
		if err != nil {
			return errs.Wrap(err, "error listing backlog items")
		}
		return nil
	})
	if err != nil {
		return result, count, err
	}
	return result, count, nil
}

func countBacklogItems(ctx context.Context, db application.DB, spaceID uuid.UUID) (int, error) {
	count := 0
	backlogExp, err := generateBacklogExpression(ctx, db, spaceID, nil)
	if err != nil || backlogExp == nil {
		return count, err
	}

	err = application.Transactional(db, func(appl application.Application) error {
		// Get the list of work items for the following criteria
		count, err = appl.WorkItems().Count(ctx, spaceID, backlogExp)
		if err != nil {
			return errs.Wrap(err, "error listing backlog items")
		}
		return nil
	})
	if err != nil {
		return count, err
	}

	return count, nil
}
