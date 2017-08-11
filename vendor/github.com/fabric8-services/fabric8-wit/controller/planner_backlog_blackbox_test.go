package controller_test

import (
	"testing"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestPlannerBacklogBlackboxREST struct {
	gormtestsupport.DBTestSuite
	clean        func()
	testIdentity account.Identity
	ctx          context.Context
}

func TestRunPlannerBacklogBlackboxREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(TestPlannerBacklogBlackboxREST))
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *TestPlannerBacklogBlackboxREST) SetupSuite() {
	rest.DBTestSuite.SetupSuite()
	rest.ctx = migration.NewMigrationContext(context.Background())
	rest.DBTestSuite.PopulateDBTestSuite(rest.ctx)
}

func (rest *TestPlannerBacklogBlackboxREST) SetupTest() {
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestPlannerBacklogBlackboxREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = *testIdentity
}

func (rest *TestPlannerBacklogBlackboxREST) TearDownTest() {
	rest.clean()
}

func (rest *TestPlannerBacklogBlackboxREST) UnSecuredController() (*goa.Service, *PlannerBacklogController) {
	svc := goa.New("PlannerBacklog-Service")
	return svc, NewPlannerBacklogController(svc, gormapplication.NewGormDB(rest.DB), rest.Configuration)
}

func (rest *TestPlannerBacklogBlackboxREST) setupPlannerBacklogWorkItems() (testSpace *space.Space, parentIteration *iteration.Iteration, createdWI *workitem.WorkItem) {
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		spacesRepo := app.Spaces()
		testSpace = &space.Space{
			Name: "PlannerBacklogWorkItems-" + uuid.NewV4().String(),
		}
		_, err := spacesRepo.Create(rest.ctx, testSpace)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), testSpace.ID)
		log.Info(nil, map[string]interface{}{"space_id": testSpace.ID}, "created space")
		workitemTypesRepo := app.WorkItemTypes()
		workitemType, err := workitemTypesRepo.Create(rest.ctx, testSpace.ID, nil, &workitem.SystemPlannerItem, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
		require.Nil(rest.T(), err)
		log.Info(nil, map[string]interface{}{"wit_id": workitemType.ID}, "created workitem type")

		iterationsRepo := app.Iterations()
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: testSpace.ID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)
		log.Info(nil, map[string]interface{}{"parent_iteration_id": parentIteration.ID}, "created parent iteration")

		childIteration := &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: testSpace.ID,
			Path:    append(parentIteration.Path, parentIteration.ID),
			State:   iteration.IterationStateStart,
		}
		iterationsRepo.Create(rest.ctx, childIteration)
		log.Info(nil, map[string]interface{}{"child_iteration_id": childIteration.ID}, "created child iteration")

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields, rest.testIdentity.ID)

		fields2 := map[string]interface{}{
			workitem.SystemTitle:     "childIteration Test",
			workitem.SystemState:     "closed",
			workitem.SystemIteration: childIteration.ID.String(),
		}
		createdWI, err = app.WorkItems().Create(rest.ctx, testSpace.ID, workitemType.ID, fields2, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
		return nil
	})
	return
}

func assertPlannerBacklogWorkItems(t *testing.T, workitems *app.WorkItemList, testSpace *space.Space, parentIteration *iteration.Iteration) {
	// Two iteration have to be found
	require.NotNil(t, workitems)
	assert.Len(t, workitems.Data, 1)
	for _, workItem := range workitems.Data {
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, testSpace.ID.String(), workItem.Relationships.Space.Data.ID.String())
		assert.Equal(t, "parentIteration Test", workItem.Attributes[workitem.SystemTitle])
		assert.Equal(t, "new", workItem.Attributes[workitem.SystemState])
		assert.Equal(t, parentIteration.ID.String(), *workItem.Relationships.Iteration.Data.ID)
	}
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOK() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfModifiedSinceHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	rest.T().Log("Test Space: " + testSpace.ID.String())
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifModifiedSince := app.ToHTTPTime(parentIteration.UpdatedAt.Add(-1 * time.Hour))
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsOkUsingExpiredIfNoneMatchHeader() {
	// given
	testSpace, parentIteration, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifNoneMatch := "foo"
	res, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertPlannerBacklogWorkItems(rest.T(), workitems, testSpace, parentIteration)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfModifiedSinceHeader() {
	// given
	testSpace, _, lastWorkItem := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	// when
	offset := "0"
	filter := ""
	limit := -1
	ifModifiedSince := app.ToHTTPTime(lastWorkItem.Fields[workitem.SystemUpdatedAt].(time.Time))
	res := test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestListPlannerBacklogWorkItemsNotModifiedUsingIfNoneMatchHeader() {
	// given
	testSpace, _, _ := rest.setupPlannerBacklogWorkItems()
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	filter := ""
	limit := -1
	res, _ := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	res = test.ListPlannerBacklogNotModified(rest.T(), svc.Context, svc, ctrl, testSpace.ID, &filter, nil, nil, nil, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestPlannerBacklogBlackboxREST) TestSuccessEmptyListPlannerBacklogWorkItems() {
	var spaceID uuid.UUID
	var parentIteration *iteration.Iteration
	application.Transactional(gormapplication.NewGormDB(rest.DB), func(app application.Application) error {
		iterationsRepo := app.Iterations()
		newSpace := space.Space{
			Name: "TestSuccessEmptyListPlannerBacklogWorkItems" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(rest.ctx, &newSpace)
		if err != nil {
			rest.T().Error(err)
		}
		spaceID = p.ID
		parentIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
			State:   iteration.IterationStateNew,
		}
		iterationsRepo.Create(rest.ctx, parentIteration)

		fields := map[string]interface{}{
			workitem.SystemTitle:     "parentIteration Test",
			workitem.SystemState:     "new",
			workitem.SystemIteration: parentIteration.ID.String(),
		}
		app.WorkItems().Create(rest.ctx, spaceID, workitem.SystemPlannerItem, fields, rest.testIdentity.ID)

		return nil
	})

	svc, ctrl := rest.UnSecuredController()

	offset := "0"
	filter := ""
	limit := -1
	_, workitems := test.ListPlannerBacklogOK(rest.T(), svc.Context, svc, ctrl, spaceID, &filter, nil, nil, nil, &limit, &offset, nil, nil)
	// The list has to be empty
	assert.Len(rest.T(), workitems.Data, 0)
}
