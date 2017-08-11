package controller_test

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"context"

	"fmt"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceIterationREST struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	clean        func()
	ctx          context.Context
	testIdentity account.Identity
}

func TestRunSpaceIterationREST(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestSpaceIterationREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (rest *TestSpaceIterationREST) SetupSuite() {
	rest.DBTestSuite.SetupSuite()
	rest.ctx = migration.NewMigrationContext(context.Background())
	rest.DBTestSuite.PopulateDBTestSuite(rest.ctx)
}

func (rest *TestSpaceIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestSpaceIterationREST user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = *testIdentity
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (rest *TestSpaceIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceIterationREST) SecuredController() (*goa.Service, *SpaceIterationsController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *SpaceIterationsController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", wittoken.NewManagerWithPrivateKey(priv), *idn)
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) UnSecuredController() (*goa.Service, *SpaceIterationsController) {
	svc := goa.New("Iteration-Service")
	return svc, NewSpaceIterationsController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIteration() {
	// given
	var p *space.Space
	var rootItr *iteration.Iteration
	ci := createSpaceIteration("Sprint #21", nil)
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := space.Space{
			Name:    "TestSuccessCreateIteration" + uuid.NewV4().String(),
			OwnerId: testsupport.TestIdentity.ID,
		}
		createdSpace, err := repo.Create(rest.ctx, &newSpace)
		p = createdSpace
		if err != nil {
			return err
		}
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: newSpace.ID,
			Name:    newSpace.Name,
		}
		iterationRepo := app.Iterations()
		err = iterationRepo.Create(rest.ctx, rootItr)
		return err
	})
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	// then
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), iteration.IterationStateNew, *c.Data.Attributes.State)
	assert.Equal(rest.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
	require.NotNil(rest.T(), c.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestSpaceIterationREST) TestSuccessCreateIterationWithOptionalValues() {
	// given
	var p *space.Space
	var rootItr *iteration.Iteration
	iterationName := "Sprint #22"
	iterationDesc := "testing description"
	ci := createSpaceIteration(iterationName, &iterationDesc)
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		testSpace := space.Space{
			Name:    "TestSuccessCreateIterationWithOptionalValues-" + uuid.NewV4().String(),
			OwnerId: testsupport.TestIdentity.ID,
		}
		p, _ = repo.Create(rest.ctx, &testSpace)
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: testSpace.ID,
			Name:    testSpace.Name,
		}
		iterationRepo := app.Iterations()
		err := iterationRepo.Create(rest.ctx, rootItr)
		require.Nil(rest.T(), err)
		return nil
	})
	svc, ctrl := rest.SecuredController()
	// when
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	// then
	assert.NotNil(rest.T(), c.Data.ID)
	assert.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName)
	assert.Equal(rest.T(), *c.Data.Attributes.Description, iterationDesc)

	// create another Iteration with nil description
	iterationName2 := "Sprint #23"
	ci = createSpaceIteration(iterationName2, nil)
	_, c = test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	assert.Equal(rest.T(), *c.Data.Attributes.Name, iterationName2)
	assert.Nil(rest.T(), c.Data.Attributes.Description)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOK() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfModifiedSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idModifiedSince := app.ToHTTPTime(fatherIteration.UpdatedAt.Add(-1 * time.Hour))
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceOKUsingExpiredIfNoneMatchSinceHeader() {
	// given
	spaceID, fatherIteration, childIteration, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when
	idNoneMatch := "foo"
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
	// then
	assertIterations(rest.T(), cs.Data, fatherIteration, childIteration, grandChildIteration)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfModifiedSinceHeader() {
	// given
	spaceID, _, _, grandChildIteration := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// when/then
	idModifiedSince := app.ToHTTPTime(grandChildIteration.UpdatedAt)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID, &idModifiedSince, nil)
}

func (rest *TestSpaceIterationREST) TestListIterationsBySpaceNotModifiedUsingIfNoneMatchSinceHeader() {
	// given
	spaceID, _, _, _ := rest.createIterations()
	svc, ctrl := rest.UnSecuredController()
	// here we need to get all iterations for the spaceId
	_, iterations := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceID, nil, nil)
	// when/then
	idNoneMatch := generateIterationsTag(*iterations)
	test.ListSpaceIterationsNotModified(rest.T(), svc.Context, svc, ctrl, spaceID, nil, &idNoneMatch)
}

func (rest *TestSpaceIterationREST) TestCreateIterationMissingSpace() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceIterationREST) TestFailCreateIterationNotAuthorized() {
	// given
	ci := createSpaceIteration("Sprint #21", nil)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateSpaceIterationsUnauthorized(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), ci)
}

func (rest *TestSpaceIterationREST) TestFailListIterationsByMissingSpace() {
	// given
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.ListSpaceIterationsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), nil, nil)
}

// Following is behaviour of the test that verifies the WI Count in an iteration
// Consider, iteration i1 has 2 children c1 & c2
// Total WI for i1 = WI assigned to i1 + WI assigned to c1 + WI assigned to c2
// Begin test with following setup :-
// Create a space s1
// create iteartion i1 & iteration i2 in s1
// Create child of i2 : name it child
// Create child of child : name it grandChild
// Add few "new" & "closed" work items to i1
// Add few "new" work items to child
// Add few "closed" work items to grandChild
// Call List-Iterations API, should return Total & Closed WI count for every itearion
// Verify counts for all 4 iterations retrieved.
// Add few "new" & "closed" work items to i2
// Call List-Iterations API, should return Total & Closed WI count for every itearion
// Verify updated count values for all 4 iterations retrieved.
func (rest *TestSpaceIterationREST) TestWICountsWithIterationListBySpace() {
	// given
	resource.Require(rest.T(), resource.Database)
	// create seed data
	spaceRepo := space.NewRepository(rest.DB)
	spaceInstance := space.Space{
		Name: "TestWICountsWithIterationListBySpace-" + uuid.NewV4().String(),
	}
	_, e := spaceRepo.Create(rest.ctx, &spaceInstance)
	require.Nil(rest.T(), e)
	require.NotEqual(rest.T(), uuid.UUID{}, spaceInstance.ID)

	iterationRepo := iteration.NewIterationRepository(rest.DB)
	iteration1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(rest.ctx, &iteration1)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration1.ID)

	iteration2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: spaceInstance.ID,
	}
	iterationRepo.Create(rest.ctx, &iteration2)
	assert.NotEqual(rest.T(), uuid.UUID{}, iteration2.ID)

	childOfIteration2 := iteration.Iteration{
		Name:    "Sprint 2.1",
		SpaceID: spaceInstance.ID,
		Path:    append(iteration2.Path, iteration2.ID),
	}
	iterationRepo.Create(rest.ctx, &childOfIteration2)
	require.NotEqual(rest.T(), uuid.Nil, childOfIteration2.ID)

	grandChildOfIteration2 := iteration.Iteration{
		Name:    "Sprint 2.1.1",
		SpaceID: spaceInstance.ID,
		Path:    append(childOfIteration2.Path, childOfIteration2.ID),
	}
	iterationRepo.Create(rest.ctx, &grandChildOfIteration2)
	require.NotEqual(rest.T(), uuid.UUID{}, grandChildOfIteration2.ID)

	wirepo := workitem.NewWorkItemRepository(rest.DB)

	for i := 0; i < 3; i++ {
		wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration1.ID.String(),
			}, rest.testIdentity.ID)
	}
	for i := 0; i < 2; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration1.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	// add items to nested iteration level 1
	for i := 0; i < 4; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: childOfIteration2.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	// add items to nested iteration level 2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: grandChildOfIteration2.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}

	svc, ctrl := rest.UnSecuredController()
	// when
	_, cs := test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceInstance.ID, nil, nil)
	// then
	require.Len(rest.T(), cs.Data, 4)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(rest.T(), 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 0 + 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 0 + 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, childOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 4 + 5  // sum of all items of self and child
			expectedClosed := 0 + 5 // sum of closed items of self and child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, grandChildOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 5 + 0  // sum of all items of self and child
			expectedClosed := 5 + 0 // sum of closed items of self and child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		}
	}
	// seed 5 New WI to iteration2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: iteration2.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	// seed 2 Closed WI to iteration2
	for i := 0; i < 3; i++ {
		_, err := wirepo.Create(
			rest.ctx, iteration1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: iteration2.ID.String(),
			}, rest.testIdentity.ID)
		require.Nil(rest.T(), err)
	}
	// when
	_, cs = test.ListSpaceIterationsOK(rest.T(), svc.Context, svc, ctrl, spaceInstance.ID, nil, nil)
	// then
	require.Len(rest.T(), cs.Data, 4)
	for _, iterationItem := range cs.Data {
		if uuid.Equal(*iterationItem.ID, iteration1.ID) {
			assert.Equal(rest.T(), 5, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), 2, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, iteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 8 + 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 3 + 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, childOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 4 + 5  // sum of all items of self + child + grand-child
			expectedClosed := 0 + 5 // sum of closed items self + child + grand-child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		} else if uuid.Equal(*iterationItem.ID, grandChildOfIteration2.ID) {
			// we expect these counts should include that of child iterations too.
			expectedTotal := 5 + 0  // sum of all items of self + child + grand-child
			expectedClosed := 5 + 0 // sum of closed items self + child + grand-child
			assert.Equal(rest.T(), expectedTotal, iterationItem.Relationships.Workitems.Meta["total"])
			assert.Equal(rest.T(), expectedClosed, iterationItem.Relationships.Workitems.Meta["closed"])
		}
	}
}

func (rest *TestSpaceIterationREST) TestOnlySpaceOwnerCreateIteration() {
	var p *space.Space
	var rootItr *iteration.Iteration
	identityRepo := account.NewIdentityRepository(rest.DB)
	spaceOwner := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "space-owner-identity",
		ProviderType: account.KeycloakIDP}
	errInCreateOwner := identityRepo.Create(rest.ctx, spaceOwner)
	require.Nil(rest.T(), errInCreateOwner)

	ci := createSpaceIteration("Sprint #21", nil)
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Spaces()
		newSpace := space.Space{
			Name:    "TestSuccessCreateIteration" + uuid.NewV4().String(),
			OwnerId: spaceOwner.ID,
		}
		createdSpace, err := repo.Create(rest.ctx, &newSpace)
		p = createdSpace
		if err != nil {
			return err
		}
		// create Root iteration for above space
		rootItr = &iteration.Iteration{
			SpaceID: newSpace.ID,
			Name:    newSpace.Name,
		}
		iterationRepo := app.Iterations()
		err = iterationRepo.Create(rest.ctx, rootItr)
		return err
	})
	require.Nil(rest.T(), err)

	spaceOwner, errInLoad := identityRepo.Load(rest.ctx, p.OwnerId)
	require.Nil(rest.T(), errInLoad)

	svc, ctrl := rest.SecuredControllerWithIdentity(spaceOwner)

	// try creating iteration with space-owner. should pass
	_, c := test.CreateSpaceIterationsCreated(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
	require.NotNil(rest.T(), c.Data.ID)
	require.NotNil(rest.T(), c.Data.Relationships.Space)
	assert.Equal(rest.T(), p.ID.String(), *c.Data.Relationships.Space.Data.ID)
	assert.Equal(rest.T(), iteration.IterationStateNew, *c.Data.Attributes.State)
	assert.Equal(rest.T(), "/"+rootItr.ID.String(), *c.Data.Attributes.ParentPath)
	require.NotNil(rest.T(), c.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, c.Data.Relationships.Workitems.Meta["closed"])

	otherIdentity := &account.Identity{
		ID:           uuid.NewV4(),
		Username:     "non-space-owner-identity",
		ProviderType: account.KeycloakIDP}
	errInCreateOther := identityRepo.Create(rest.ctx, otherIdentity)
	require.Nil(rest.T(), errInCreateOther)

	svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
	test.CreateSpaceIterationsForbidden(rest.T(), svc.Context, svc, ctrl, p.ID, ci)
}

func createSpaceIteration(name string, desc *string) *app.CreateSpaceIterationsPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	return &app.CreateSpaceIterationsPayload{
		Data: &app.Iteration{
			Type: iteration.APIStringTypeIteration,
			Attributes: &app.IterationAttributes{
				Name:        &name,
				StartAt:     &start,
				EndAt:       &end,
				Description: desc,
			},
		},
	}
}

func (rest *TestSpaceIterationREST) createIterations() (spaceID uuid.UUID, fatherIteration, childIteration, grandChildIteration *iteration.Iteration) {
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()
		newSpace := space.Space{
			Name: "TestListIterationsBySpace-" + uuid.NewV4().String(),
		}
		p, err := app.Spaces().Create(rest.ctx, &newSpace)
		if err != nil {
			return err
		}
		spaceID = p.ID
		for i := 0; i < 3; i++ {
			start := time.Now()
			end := start.Add(time.Hour * (24 * 8 * 3))
			name := "Sprint Test #" + strconv.Itoa(i)
			i := iteration.Iteration{
				Name:    name,
				SpaceID: spaceID,
				StartAt: &start,
				EndAt:   &end,
			}
			repo.Create(rest.ctx, &i)
		}
		// create one child iteration and test for relationships.Parent
		fatherIteration = &iteration.Iteration{
			Name:    "Parent Iteration",
			SpaceID: spaceID,
		}
		repo.Create(rest.ctx, fatherIteration)
		rest.T().Log("fatherIteration:", fatherIteration.ID, fatherIteration.Name, fatherIteration.Path)
		childIteration = &iteration.Iteration{
			Name:    "Child Iteration",
			SpaceID: spaceID,
			Path:    append(fatherIteration.Path, fatherIteration.ID),
		}
		repo.Create(rest.ctx, childIteration)
		rest.T().Log("childIteration:", childIteration.ID, childIteration.Name, childIteration.Path)
		grandChildIteration = &iteration.Iteration{
			Name:    "Grand Child Iteration",
			SpaceID: spaceID,
			Path:    append(childIteration.Path, childIteration.ID),
		}
		repo.Create(rest.ctx, grandChildIteration)
		rest.T().Log("grandChildIteration:", grandChildIteration.ID, grandChildIteration.Name, grandChildIteration.Path)

		return nil
	})
	require.Nil(rest.T(), err)
	return
}

func assertIterations(t *testing.T, data []*app.Iteration, fatherIteration, childIteration, grandChildIteration *iteration.Iteration) {
	assert.Len(t, data, 6)
	for _, iterationItem := range data {
		subString := fmt.Sprintf("?filter[iteration]=%s", iterationItem.ID.String())
		require.Contains(t, *iterationItem.Relationships.Workitems.Links.Related, subString)
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["total"])
		assert.Equal(t, 0, iterationItem.Relationships.Workitems.Meta["closed"])
		if *iterationItem.ID == childIteration.ID {
			t.Log("childIteration:", iterationItem.ID, *iterationItem.Attributes.Name, *iterationItem.Attributes.ParentPath, *iterationItem.Relationships.Parent.Data.ID)
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, fatherIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)
		}
		if *iterationItem.ID == grandChildIteration.ID {
			t.Log("grandChildIteration:", iterationItem.ID, *iterationItem.Attributes.Name, *iterationItem.Attributes.ParentPath, *iterationItem.Relationships.Parent.Data.ID)
			expectedParentPath := iteration.PathSepInService + fatherIteration.ID.String() + iteration.PathSepInService + childIteration.ID.String()
			expectedResolvedParentPath := iteration.PathSepInService + fatherIteration.Name + iteration.PathSepInService + childIteration.Name
			require.NotNil(t, iterationItem.Relationships.Parent)
			assert.Equal(t, childIteration.ID.String(), *iterationItem.Relationships.Parent.Data.ID)
			assert.Equal(t, expectedParentPath, *iterationItem.Attributes.ParentPath)
			assert.Equal(t, expectedResolvedParentPath, *iterationItem.Attributes.ResolvedParentPath)

		}
	}
}

func generateIterationsTag(iterations app.IterationList) string {
	modelEntities := make([]app.ConditionalRequestEntity, len(iterations.Data))
	for i, entity := range iterations.Data {
		modelEntities[i] = iteration.Iteration{
			ID: *entity.ID,
			Lifecycle: gormsupport.Lifecycle{
				UpdatedAt: *entity.Attributes.UpdatedAt,
			},
		}
	}
	return app.GenerateEntitiesTag(modelEntities)
}
