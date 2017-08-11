package controller_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"context"

	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestIterationREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunIterationREST(t *testing.T) {
	// given
	suite.Run(t, &TestIterationREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestIterationREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestIterationREST) TearDownTest() {
	rest.clean()
}

func (rest *TestIterationREST) SecuredController() (*goa.Service, *IterationController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewIterationController(svc, rest.db, rest.Configuration)
}

func (rest *TestIterationREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *IterationController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Iteration-Service", wittoken.NewManagerWithPrivateKey(priv), *idn)
	return svc, NewIterationController(svc, rest.db, rest.Configuration)
}

func (rest *TestIterationREST) UnSecuredController() (*goa.Service, *IterationController) {
	svc := goa.New("Iteration-Service")
	return svc, NewIterationController(svc, rest.db, rest.Configuration)
}

func (rest *TestIterationREST) TestSuccessCreateChildIteration() {
	// given
	sp, _, _, _, parent := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	ri, err := rest.db.Iterations().Root(context.Background(), parent.SpaceID)
	require.Nil(rest.T(), err)
	parentID := parent.ID
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, created := test.CreateChildIterationCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	require.NotNil(rest.T(), created)
	assertChildIterationLinking(rest.T(), created.Data)
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	expectedParentPath := parent.Path.String() + path.SepInService + parentID.String()
	expectedResolvedParentPath := path.SepInService + ri.Name + path.SepInService + parent.Name
	assert.Equal(rest.T(), expectedParentPath, *created.Data.Attributes.ParentPath)
	assert.Equal(rest.T(), expectedResolvedParentPath, *created.Data.Attributes.ResolvedParentPath)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])

	// try to create child iteration with some other user
	otherIdentity := &account.Identity{
		Username:     "non-space-owner-identity",
		ProviderType: account.KeycloakIDP,
	}
	errInCreateOther := rest.db.Identities().Create(context.Background(), otherIdentity)
	require.Nil(rest.T(), errInCreateOther)
	svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
	test.CreateChildIterationForbidden(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestFailCreateSameChildIterationConflict() {
	// given
	sp, _, _, _, parent := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	ri, err := rest.db.Iterations().Root(context.Background(), parent.SpaceID)
	require.Nil(rest.T(), err)
	parentID := parent.ID
	name := uuid.NewV4().String()
	ci := getChildIterationPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, created := test.CreateChildIterationCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	require.NotNil(rest.T(), created)
	assertChildIterationLinking(rest.T(), created.Data)
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	expectedParentPath := parent.Path.String() + path.SepInService + parentID.String()
	expectedResolvedParentPath := path.SepInService + ri.Name + path.SepInService + parent.Name
	assert.Equal(rest.T(), expectedParentPath, *created.Data.Attributes.ParentPath)
	assert.Equal(rest.T(), expectedResolvedParentPath, *created.Data.Attributes.ResolvedParentPath)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])

	// try creating again with same name + hierarchy
	test.CreateChildIterationConflict(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestIterationREST) TestFailValidationIterationNameLength() {
	// given
	_, _, _, _, parent := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	_, err := rest.db.Iterations().Root(context.Background(), parent.SpaceID)
	require.Nil(rest.T(), err)
	ci := getChildIterationPayload(&testsupport.TestOversizedNameObj)

	err = ci.Validate()
	// Validate payload function returns an error
	assert.NotNil(rest.T(), err)
	assert.Contains(rest.T(), err.Error(), "length of response.name must be less than or equal to than 62")
}

func (rest *TestIterationREST) TestFailValidationIterationNameStartWith() {
	// given
	_, _, _, _, parent := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	_, err := rest.db.Iterations().Root(context.Background(), parent.SpaceID)
	require.Nil(rest.T(), err)
	name := "_Sprint #21"
	ci := getChildIterationPayload(&name)

	err = ci.Validate()
	// Validate payload function returns an error
	assert.NotNil(rest.T(), err)
	assert.Contains(rest.T(), err.Error(), "response.name must match the regexp")
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingName() {
	sp, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	ci := getChildIterationPayload(nil)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	test.CreateChildIterationBadRequest(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationMissingParent() {
	// given
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildIterationNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), ci)
}

func (rest *TestIterationREST) TestFailCreateChildIterationNotAuthorized() {
	// when
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	name := "Sprint #21"
	ci := getChildIterationPayload(&name)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateChildIterationUnauthorized(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), ci)
}

func (rest *TestIterationREST) TestShowIterationOK() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), nil, nil)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSinceHeader := app.ToHTTPTime(itr.UpdatedAt.Add(-1 * time.Hour))
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &ifModifiedSinceHeader, nil)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := "foo"
	_, created := test.ShowIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), nil, &ifNoneMatch)
	// then
	assertIterationLinking(rest.T(), created.Data)
	require.NotNil(rest.T(), created.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, created.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestShowIterationNotModifiedUsingIfModifiedSinceHeader() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when/then
	rest.T().Log("Iteration:", itr, " updatedAt: ", itr.UpdatedAt)
	ifModifiedSinceHeader := app.ToHTTPTime(itr.UpdatedAt)
	test.ShowIterationNotModified(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &ifModifiedSinceHeader, nil)
}

func (rest *TestIterationREST) TestShowIterationNotModifiedUsingIfNoneMatchHeader() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when/then
	ifNoneMatch := app.GenerateEntityTag(itr)
	test.ShowIterationNotModified(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), nil, &ifNoneMatch)
}

func (rest *TestIterationREST) TestFailShowIterationMissing() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	test.ShowIterationNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestIterationREST) TestSuccessUpdateIteration() {
	// given
	sp, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	newName := "Sprint 1001"
	newDesc := "New Description"
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				Name:        &newName,
				Description: &newDesc,
			},
			ID:   &itr.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	owner, errIdn := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), errIdn)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
	// then
	assert.Equal(rest.T(), newName, *updated.Data.Attributes.Name)
	assert.Equal(rest.T(), newDesc, *updated.Data.Attributes.Description)
	require.NotNil(rest.T(), updated.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 0, updated.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 0, updated.Data.Relationships.Workitems.Meta["closed"])

	// try update using some other user
	otherIdentity := &account.Identity{
		Username:     "non-space-owner-identity",
		ProviderType: account.KeycloakIDP,
	}
	errInCreateOther := rest.db.Identities().Create(context.Background(), otherIdentity)
	require.Nil(rest.T(), errInCreateOther)
	svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
	test.UpdateIterationForbidden(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestSuccessUpdateIterationWithWICounts() {
	// given
	sp, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	newName := "Sprint 1001"
	newDesc := "New Description"
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				Name:        &newName,
				Description: &newDesc,
			},
			ID:   &itr.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	// add WI to this iteration and test counts in the response of update iteration API
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestSuccessUpdateIterationWithWICounts user", "test provider")
	require.Nil(rest.T(), err)
	wirepo := workitem.NewWorkItemRepository(rest.DB)
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	for i := 0; i < 4; i++ {
		wi, err := wirepo.Create(
			ctx, itr.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateNew,
				workitem.SystemIteration: itr.ID.String(),
			}, testIdentity.ID)
		require.NotNil(rest.T(), wi)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), wi)
	}
	for i := 0; i < 5; i++ {
		wi, err := wirepo.Create(
			ctx, itr.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: itr.ID.String(),
			}, testIdentity.ID)
		require.NotNil(rest.T(), wi)
		require.Nil(rest.T(), err)
		require.NotNil(rest.T(), wi)
	}
	owner, errIdn := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), errIdn)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
	// then
	require.NotNil(rest.T(), updated)
	assert.Equal(rest.T(), newName, *updated.Data.Attributes.Name)
	assert.Equal(rest.T(), newDesc, *updated.Data.Attributes.Description)
	require.NotNil(rest.T(), updated.Data.Relationships.Workitems.Meta)
	assert.Equal(rest.T(), 9, updated.Data.Relationships.Workitems.Meta["total"])
	assert.Equal(rest.T(), 5, updated.Data.Relationships.Workitems.Meta["closed"])
}

func (rest *TestIterationREST) TestFailUpdateIterationNotFound() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	itr.ID = uuid.NewV4()
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.SecuredController()
	// when/then
	test.UpdateIterationNotFound(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestFailUpdateIterationUnauthorized() {
	// given
	_, _, _, _, itr := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{},
			ID:         &itr.ID,
			Type:       iteration.APIStringTypeIteration,
		},
	}
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.UpdateIterationUnauthorized(rest.T(), svc.Context, svc, ctrl, itr.ID.String(), &payload)
}

func (rest *TestIterationREST) TestIterationStateTransitions() {
	// given
	sp, _, _, _, itr1 := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	assert.Equal(rest.T(), iteration.IterationStateNew, itr1.State)
	startState := iteration.IterationStateStart
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr1.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	owner, errIdn := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), errIdn)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	_, updated := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(rest.T(), startState, *updated.Data.Attributes.State)
	// create another iteration in same space and then change State to start
	itr2 := iteration.Iteration{
		Name:    "Spring 123",
		SpaceID: itr1.SpaceID,
		Path:    itr1.Path,
	}
	err := rest.db.Iterations().Create(context.Background(), &itr2)
	require.Nil(rest.T(), err)
	payload2 := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &itr2.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	test.UpdateIterationBadRequest(rest.T(), svc.Context, svc, ctrl, itr2.ID.String(), &payload2)
	// now close first iteration
	closeState := iteration.IterationStateClose
	payload.Data.Attributes.State = &closeState
	_, updated = test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr1.ID.String(), &payload)
	assert.Equal(rest.T(), closeState, *updated.Data.Attributes.State)
	// try to start iteration 2 now
	_, updated2 := test.UpdateIterationOK(rest.T(), svc.Context, svc, ctrl, itr2.ID.String(), &payload2)
	assert.Equal(rest.T(), startState, *updated2.Data.Attributes.State)
}

func (rest *TestIterationREST) TestRootIterationCanNotStart() {
	// given
	sp, _, _, _, itr1 := createSpaceAndRootAreaAndIterations(rest.T(), rest.db)
	var ri *iteration.Iteration
	err := application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Iterations()
		var err error
		ri, err = repo.Root(context.Background(), itr1.SpaceID)
		return err
	})
	require.Nil(rest.T(), err)
	require.NotNil(rest.T(), ri)

	startState := iteration.IterationStateStart
	payload := app.UpdateIterationPayload{
		Data: &app.Iteration{
			Attributes: &app.IterationAttributes{
				State: &startState,
			},
			ID:   &ri.ID,
			Type: iteration.APIStringTypeIteration,
		},
	}
	owner, errIdn := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), errIdn)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	test.UpdateIterationBadRequest(rest.T(), svc.Context, svc, ctrl, ri.ID.String(), &payload)
}

func getChildIterationPayload(name *string) *app.CreateChildIterationPayload {
	start := time.Now()
	end := start.Add(time.Hour * (24 * 8 * 3))

	itType := iteration.APIStringTypeIteration

	return &app.CreateChildIterationPayload{
		Data: &app.Iteration{
			Type: itType,
			Attributes: &app.IterationAttributes{
				Name:    name,
				StartAt: &start,
				EndAt:   &end,
			},
		},
	}
}

// following helper function creates a space , root area, root iteration for that space.
// Also creates a new iteration and new area in the same space
func createSpaceAndRootAreaAndIterations(t *testing.T, db application.DB) (space.Space, area.Area, iteration.Iteration, area.Area, iteration.Iteration) {
	var (
		spaceObj          space.Space
		rootAreaObj       area.Area
		rootIterationObj  iteration.Iteration
		otherIterationObj iteration.Iteration
		otherAreaObj      area.Area
	)
	application.Transactional(db, func(app application.Application) error {
		owner := &account.Identity{
			Username:     "new-space-owner-identity",
			ProviderType: account.KeycloakIDP,
		}
		errCreateOwner := app.Identities().Create(context.Background(), owner)
		require.Nil(t, errCreateOwner)
		spaceObj = space.Space{
			Name:    testsupport.CreateRandomValidTestName("CreateSpaceAndRootAreaAndIterations-"),
			OwnerId: owner.ID,
		}
		_, err := app.Spaces().Create(context.Background(), &spaceObj)
		require.Nil(t, err)
		// create the root area
		rootAreaObj = area.Area{
			Name:    spaceObj.Name,
			SpaceID: spaceObj.ID,
		}
		err = app.Areas().Create(context.Background(), &rootAreaObj)
		require.Nil(t, err)
		// above space should have a root iteration for itself
		rootIterationObj = iteration.Iteration{
			Name:    spaceObj.Name,
			SpaceID: spaceObj.ID,
		}
		err = app.Iterations().Create(context.Background(), &rootIterationObj)
		require.Nil(t, err)
		start := time.Now()
		end := start.Add(time.Hour * (24 * 8 * 3))
		iterationName := "Sprint #2"
		otherIterationObj = iteration.Iteration{
			Lifecycle: gormsupport.Lifecycle{
				CreatedAt: spaceObj.CreatedAt,
				UpdatedAt: spaceObj.UpdatedAt,
			},
			Name:    iterationName,
			SpaceID: spaceObj.ID,
			StartAt: &start,
			EndAt:   &end,
			Path:    append(rootIterationObj.Path, rootIterationObj.ID),
		}
		err = app.Iterations().Create(context.Background(), &otherIterationObj)
		require.Nil(t, err)

		areaName := "Area #2"
		otherAreaObj = area.Area{
			Lifecycle: gormsupport.Lifecycle{
				CreatedAt: spaceObj.CreatedAt,
				UpdatedAt: spaceObj.UpdatedAt,
			},
			Name:    areaName,
			SpaceID: spaceObj.ID,
			Path:    append(rootAreaObj.Path, rootAreaObj.ID),
		}
		err = app.Areas().Create(context.Background(), &otherAreaObj)
		require.Nil(t, err)
		return nil
	})
	t.Log("Created space with ID=", spaceObj.ID.String(), "name=", spaceObj.Name)
	return spaceObj, rootAreaObj, rootIterationObj, otherAreaObj, otherIterationObj
}

func assertIterationLinking(t *testing.T, target *app.Iteration) {
	assert.NotNil(t, target.ID)
	assert.Equal(t, iteration.APIStringTypeIteration, target.Type)
	assert.NotNil(t, target.Links.Self)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Space)
	require.NotNil(t, target.Relationships.Space.Links)
	require.NotNil(t, target.Relationships.Space.Links.Self)
	assert.True(t, strings.Contains(*target.Relationships.Space.Links.Self, "/api/spaces/"))
}

func assertChildIterationLinking(t *testing.T, target *app.Iteration) {
	assertIterationLinking(t, target)
	require.NotNil(t, target.Relationships)
	require.NotNil(t, target.Relationships.Parent)
	require.NotNil(t, target.Relationships.Parent.Links)
	require.NotNil(t, target.Relationships.Parent.Links.Self)
}
