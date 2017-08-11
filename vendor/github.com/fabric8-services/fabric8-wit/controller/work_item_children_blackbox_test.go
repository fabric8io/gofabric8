package controller_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// The workItemChildSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemChildSuite struct {
	gormtestsupport.DBTestSuite

	workitemLinkTypeCtrl     *WorkItemLinkTypeController
	workitemLinkCategoryCtrl *WorkItemLinkCategoryController
	workitemLinkCtrl         *WorkItemLinkController
	workItemCtrl             *WorkitemController
	workItemsCtrl            *WorkitemsController
	workItemRelsLinksCtrl    *WorkItemRelationshipsLinksController
	spaceCtrl                *SpaceController
	svc                      *goa.Service
	typeCtrl                 *WorkitemtypeController
	// These IDs can safely be used by all tests
	bug1                 *app.WorkItemSingle
	bug2                 *app.WorkItemSingle
	bug3                 *app.WorkItemSingle
	bugBlockerLinkTypeID uuid.UUID
	userSpaceID          uuid.UUID

	// Store IDs of resources that need to be removed at the beginning or end of a test
	testIdentity account.Identity
	db           *gormapplication.GormDB
	clean        func()
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemChildSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)

	s.db = gormapplication.NewGormDB(s.DB)
}

const (
	hasChildren   bool = true
	hasNoChildren bool = false
)

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item links that we will create already exist.
// It will also make sure that some resources that we rely on do exists.
func (s *workItemChildSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "workItemChildSuite user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity

	priv, err := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	require.Nil(s.T(), err)

	svc := testsupport.ServiceAsUser("WorkItemLink-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workitemLinkCtrl = NewWorkItemLinkController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workitemLinkCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLinkType-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workitemLinkTypeCtrl = NewWorkItemLinkTypeController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workitemLinkTypeCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLinkCategory-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workitemLinkCategoryCtrl = NewWorkItemLinkCategoryController(svc, s.db)
	require.NotNil(s.T(), s.workitemLinkCategoryCtrl)

	svc = testsupport.ServiceAsUser("WorkItemType-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.typeCtrl = NewWorkitemtypeController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.typeCtrl)

	svc = testsupport.ServiceAsUser("WorkItemLink-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workitemLinkCtrl = NewWorkItemLinkController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workitemLinkCtrl)

	svc = testsupport.ServiceAsUser("WorkItemRelationshipsLinks-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.workItemRelsLinksCtrl = NewWorkItemRelationshipsLinksController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workItemRelsLinksCtrl)

	svc = testsupport.ServiceAsUser("TestWorkItem-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), svc)
	s.svc = svc
	s.workItemCtrl = NewWorkitemController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workItemCtrl)

	svc = testsupport.ServiceAsUser("TestWorkItems-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity)
	require.NotNil(s.T(), svc)
	s.svc = svc
	s.workItemsCtrl = NewWorkitemsController(svc, s.db, s.Configuration)
	require.NotNil(s.T(), s.workItemsCtrl)

	svc = testsupport.ServiceAsUser("Space-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity)
	require.NotNil(s.T(), svc)
	s.spaceCtrl = NewSpaceController(svc, s.db, s.Configuration, &DummyResourceManager{})
	require.NotNil(s.T(), s.spaceCtrl)

	// Create a test user identity
	s.svc = testsupport.ServiceAsUser("TestWorkItem-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	require.NotNil(s.T(), s.svc)

	// Create a work item link space
	createSpacePayload := CreateSpacePayload("test-space"+uuid.NewV4().String(), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, createSpacePayload)
	s.userSpaceID = *space.Data.ID
	s.T().Logf("Created link space with ID: %s\n", *space.Data.ID)

	// Create 3 work items (bug1, bug2, and bug3)
	bug1Payload := newCreateWorkItemPayload(s.userSpaceID, workitem.SystemBug, "bug1")
	_, s.bug1 = test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, bug1Payload)
	require.NotNil(s.T(), s.bug1)
	checkChildrenRelationship(s.T(), s.bug1.Data, hasNoChildren)
	s.T().Logf("Created bug1 with ID: %s\n", *s.bug1.Data.ID)

	bug2Payload := newCreateWorkItemPayload(s.userSpaceID, workitem.SystemBug, "bug2")
	_, s.bug2 = test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, bug2Payload)
	require.NotNil(s.T(), s.bug2)
	checkChildrenRelationship(s.T(), s.bug2.Data, hasNoChildren)
	s.T().Logf("Created bug2 with ID: %s\n", *s.bug2.Data.ID)

	bug3Payload := newCreateWorkItemPayload(s.userSpaceID, workitem.SystemBug, "bug3")
	_, s.bug3 = test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, bug3Payload)
	require.NotNil(s.T(), s.bug3)
	checkChildrenRelationship(s.T(), s.bug3.Data, hasNoChildren)
	s.T().Logf("Created bug3 with ID: %s\n", *s.bug3.Data.ID)

	// Create a work item link category
	description := "This work item link category is managed by an admin user."
	userLinkCategoryID := createWorkItemLinkCategoryInRepo(s.T(), s.db, s.svc.Context, link.WorkItemLinkCategory{
		Name:        "test-user",
		Description: &description,
	})
	s.T().Logf("Created link category with ID: %s\n", userLinkCategoryID)

	// Create work item link type payload
	createLinkTypePayload := createParentChildWorkItemLinkType("test-bug-blocker", userLinkCategoryID, s.userSpaceID)
	workitemLinkType := createWorkItemLinkTypeInRepo(s.T(), s.db, s.svc.Context, createLinkTypePayload)
	require.NotNil(s.T(), workitemLinkType)
	s.bugBlockerLinkTypeID = *workitemLinkType.Data.ID
	s.T().Logf("Created link type with ID: %s\n", *workitemLinkType.Data.ID)
}

func (s *workItemChildSuite) linkWorkItems(source, target *app.WorkItemSingle) app.WorkItemLinkSingle {
	createPayload := newCreateWorkItemLinkPayload(*source.Data.ID, *target.Data.ID, s.bugBlockerLinkTypeID)
	_, workitemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, createPayload)
	require.NotNil(s.T(), workitemLink)
	return *workitemLink
}

func (s *workItemChildSuite) updateWorkItemLink(workitemLinkID uuid.UUID, source, target *app.WorkItemSingle) app.WorkItemLinkSingle {
	updatePayload := newUpdateWorkItemLinkPayload(workitemLinkID, *source.Data.ID, *target.Data.ID, s.bugBlockerLinkTypeID)
	log.Info(nil, nil, fmt.Sprintf("Updating work item link from %v to %v", *source.Data.ID, *target.Data.ID))
	_, workitemLink := test.UpdateWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, workitemLinkID, updatePayload)
	require.NotNil(s.T(), workitemLink)
	return *workitemLink
}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemChildSuite) TearDownTest() {
	s.clean()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createParentChildWorkItemLinkType defines a work item link type
func createParentChildWorkItemLinkType(name string, categoryID, spaceID uuid.UUID) *app.CreateWorkItemLinkTypePayload {
	description := "Specify that one bug blocks another one."
	lt := link.WorkItemLinkType{
		Name:           name,
		Description:    &description,
		Topology:       link.TopologyTree,
		ForwardName:    "parent of",
		ReverseName:    "child of",
		LinkCategoryID: categoryID,
		SpaceID:        spaceID,
	}
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	payload := ConvertWorkItemLinkTypeFromModel(reqLong, lt)
	// The create payload is required during creation. Simply copy data over.
	return &app.CreateWorkItemLinkTypePayload{
		Data: payload.Data,
	}
}

// checkChildrenRelationship runs a variety of checks on a given work item
// regarding the children relationships
func checkChildrenRelationship(t *testing.T, wi *app.WorkItem, expectedHasChildren ...bool) {
	t.Log(fmt.Sprintf("Checking relationships for work item with id=%s", *wi.ID))
	require.NotNil(t, wi.Relationships.Children, "no 'children' relationship found in work item %s", *wi.ID)
	require.NotNil(t, wi.Relationships.Children.Links, "no 'links' found in 'children' relationship in work item %s", *wi.ID)
	require.NotNil(t, wi.Relationships.Children.Meta, "no 'meta' found in 'children' relationship in work item %s", *wi.ID)
	hasChildren, hasChildrenFound := wi.Relationships.Children.Meta["hasChildren"]
	require.True(t, hasChildrenFound, "no 'hasChildren' found in 'meta' object of 'children' relationship in work item %s", *wi.ID)
	if expectedHasChildren != nil && len(expectedHasChildren) > 0 {
		assert.Equal(t, expectedHasChildren[0], hasChildren, "work item %s is supposed to have children? %v", *wi.ID, expectedHasChildren[0])
	}
}

func assertWorkItemList(t *testing.T, workItemList *app.WorkItemList) {
	assert.Equal(t, 2, len(workItemList.Data))
	count := 0
	for _, v := range workItemList.Data {
		checkChildrenRelationship(t, v)
		switch v.Attributes[workitem.SystemTitle] {
		case "bug2":
			count = count + 1
		case "bug3":
			count = count + 1
		}
	}
	assert.Equal(t, 2, count)
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemChildren(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &workItemChildSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *workItemChildSuite) TestChildren() {
	// given
	s.linkWorkItems(s.bug1, s.bug2)
	s.linkWorkItems(s.bug1, s.bug3)

	s.T().Run("show action has children", func(t *testing.T) {
		_, workItem := test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
		checkChildrenRelationship(t, workItem.Data, hasChildren)
	})
	s.T().Run("show action has no children", func(t *testing.T) {
		_, workItem := test.ShowWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug3.Data.ID, nil, nil)
		checkChildrenRelationship(t, workItem.Data, hasNoChildren)
	})
	s.T().Run("list ok", func(t *testing.T) {
		// when
		res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, nil, nil)
		// then
		assertWorkItemList(t, workItemList)
		assertResponseHeaders(t, res)
	})
	s.T().Run("using expired if modified since header", func(t *testing.T) {
		// when
		ifModifiedSince := app.ToHTTPTime(s.bug1.Data.Attributes[workitem.SystemUpdatedAt].(time.Time).Add(-1 * time.Hour))
		res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, &ifModifiedSince, nil)
		// then
		assertWorkItemList(t, workItemList)
		assertResponseHeaders(t, res)
	})
	s.T().Run("using expired if none match header", func(t *testing.T) {
		// when
		ifNoneMatch := "foo"
		res, workItemList := test.ListChildrenWorkitemOK(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, nil, &ifNoneMatch)
		// then
		assertWorkItemList(t, workItemList)
		assertResponseHeaders(t, res)
	})
	s.T().Run("not modified using if modified since header", func(t *testing.T) {
		// when
		ifModifiedSince := app.ToHTTPTime(s.bug3.Data.Attributes[workitem.SystemUpdatedAt].(time.Time))
		res := test.ListChildrenWorkitemNotModified(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, &ifModifiedSince, nil)
		// then
		assertResponseHeaders(t, res)
	})
	s.T().Run("not modified using if none match header", func(t *testing.T) {
		res, _ := test.ListChildrenWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, nil, nil)
		// when
		ifNoneMatch := res.Header()[app.ETag][0]
		res = test.ListChildrenWorkitemNotModified(t, s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil, nil, &ifNoneMatch)
		// then
		assertResponseHeaders(t, res)
	})
}

func (s *workItemChildSuite) TestWorkItemListFilterByNoParents() {
	s.linkWorkItems(s.bug1, s.bug2)
	s.linkWorkItems(s.bug1, s.bug3)

	s.T().Run("without parentexists filter", func(t *testing.T) {
		// given
		var pe *bool
		// when
		_, result := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, pe, nil, nil, nil, nil, nil, nil)
		// then
		assert.Len(t, result.Data, 3)
	})

	s.T().Run("with parentexists value set to false", func(t *testing.T) {
		// given
		pe := false
		// when
		_, result2 := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, &pe, nil, nil, nil, nil, nil, nil)
		// then
		assert.Len(t, result2.Data, 1)
	})

	s.T().Run("with parentexists value set to true", func(t *testing.T) {
		// given
		pe := true
		// when
		_, result2 := test.ListWorkitemsOK(t, nil, nil, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, &pe, nil, nil, nil, nil, nil, nil)
		// then
		assert.Len(t, result2.Data, 3)
	})

}

// ------------------------------------------------------------------------
// Testing that the 'show' and 'list' operations return an updated list of
// work items when one of them has been linked to another one, or a link
// was updated or (soft) delete
// ------------------------------------------------------------------------

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOK() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	log.Warn(nil, map[string]interface{}{"wi_id": *s.bug1.Data.ID}, "Using ifModifiedSince=%v", ifModifiedSince)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOK() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOK() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	ifNoneMatch := "foo"
	_, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenShowOKUsingIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	// add another link
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	res, workitemSingle = test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemSingle)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)

}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOK() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenOKThenListUsingIfModifiedSinceHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasNoChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug2)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkThenListToChildrenOK() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndDeleteLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	// create a link then remove it
	workitemLink12 := s.linkWorkItems(s.bug1, s.bug2)
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	test.DeleteWorkItemLinkOK(s.T(), s.svc.Context, s.svc, s.workitemLinkCtrl, *workitemLink12.Data.ID)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasNoChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOK() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingExpiredIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingIfModifiedSinceHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when/then
	updatedAt := workitemSingle.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingExpiredIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	_, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	ifNoneMatch := "foo"
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func (s *workItemChildSuite) TestCreateAndUpdateLinkToChildrenThenListOKUsingIfNoneMatchHeader() {
	// given
	// create a link then add another one
	s.linkWorkItems(s.bug1, s.bug2)
	res, workitemSingle := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workItemCtrl, *s.bug1.Data.ID, nil, nil)
	checkChildrenRelationship(s.T(), workitemSingle.Data, hasChildren)
	time.Sleep(1 * time.Second)
	s.linkWorkItems(s.bug1, s.bug3)
	// when
	ifNoneMatch := res.Header()[app.ETag][0]
	_, workitemList := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workItemsCtrl, s.userSpaceID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	require.NotNil(s.T(), workitemList)
	checkChildrenRelationship(s.T(), lookupWorkitem(s.T(), *workitemList, *s.bug1.Data.ID), hasChildren)
}

func lookupWorkitem(t *testing.T, wiList app.WorkItemList, wiID uuid.UUID) *app.WorkItem {
	for _, wiData := range wiList.Data {
		if *wiData.ID == wiID {
			return wiData
		}
	}
	t.Error(fmt.Sprintf("Failed to look-up work item with id='%s'", wiID))
	return nil
}
