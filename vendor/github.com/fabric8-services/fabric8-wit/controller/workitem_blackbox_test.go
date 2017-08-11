package controller_test

import (
	"bytes"
	"crypto/rsa"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const none = "none"

func TestSuiteWorkItem1(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(WorkItemSuite))
}

type WorkItemSuite struct {
	gormtestsupport.DBTestSuite
	clean          func()
	workitemCtrl   app.WorkitemController
	workitemsCtrl  app.WorkitemsController
	spaceCtrl      app.SpaceController
	pubKey         *rsa.PublicKey
	priKey         *rsa.PrivateKey
	svc            *goa.Service
	wi             *app.WorkItem
	minimumPayload *app.UpdateWorkitemPayload
	testIdentity   account.Identity
	ctx            context.Context
	repoWit        workitem.WorkItemRepository
}

func (s *WorkItemSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
	s.priKey, _ = wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	s.repoWit = workitem.NewWorkItemRepository(s.DB)
}

func (s *WorkItemSuite) TearDownSuite() {
	if s.DB != nil {
		s.DB.Close()
	}
}

func (s *WorkItemSuite) TearDownTest() {
	s.clean()
}

func (s *WorkItemSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "WorkItemSuite setup user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity

	s.svc = testsupport.ServiceAsUser("TestUpdateWI-Service", wittoken.NewManagerWithPrivateKey(s.priKey), s.testIdentity)
	s.workitemCtrl = NewWorkitemController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.workitemsCtrl = NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	log.Info(nil, nil, "Creating work item during test setup...")
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	log.Info(nil, nil, "Creating work item during test setup: done")
	s.wi = wi.Data
	s.minimumPayload = getMinimumRequiredUpdatePayload(s.wi)
}

func (s *WorkItemSuite) TestPagingLinks() {
	workitemsCtrl := NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	// With only ONE work item
	pagingTest := createPagingTest(s.T(), s.svc.Context, workitemsCtrl, &s.repoWit, space.SystemSpace, 1)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=2", "")

	// With only TEN work items
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	for i := 1; i <= 9; i++ {
		payload.Data.Attributes[workitem.SystemTitle] = fmt.Sprintf("Paging WI %d", i)
		test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	}
	pagingTest = createPagingTest(s.T(), s.svc.Context, workitemsCtrl, &s.repoWit, space.SystemSpace, 10)
	pagingTest(2, 5, "page[offset]=0&page[limit]=2", "page[offset]=7&page[limit]=5", "page[offset]=0&page[limit]=2", "page[offset]=7&page[limit]=5")
	pagingTest(1, 10, "page[offset]=0&page[limit]=1", "page[offset]=1&page[limit]=10", "page[offset]=0&page[limit]=1", "")
	pagingTest(0, 4, "page[offset]=0&page[limit]=4", "page[offset]=8&page[limit]=4", "", "page[offset]=4&page[limit]=4")

	// With only ZERO work items
	spaceName := "paging zero space " + uuid.NewV4().String()
	sp := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name: &spaceName,
			},
		},
	}
	_, customSpace := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, sp)
	pagingTest = createPagingTest(s.T(), s.svc.Context, workitemsCtrl, &s.repoWit, *customSpace.Data.ID, 0)
	pagingTest(10, 2, "page[offset]=0&page[limit]=0", "page[offset]=0&page[limit]=0", "", "")
	pagingTest(0, 2, "page[offset]=0&page[limit]=2", "page[offset]=0&page[limit]=0", "", "")
}

func (s *WorkItemSuite) TestPagingErrors() {
	var offset string = "-1"
	var limit int = 2
	_, result := test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "0"
	limit = 0
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is 0", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "0"
	limit = -1
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}

	offset = "-3"
	limit = -1
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is negative", "Expected limit to be default size %d, but was %s", 20, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}

	offset = "ALPHA"
	limit = 40
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	if !strings.Contains(*result.Links.First, "page[limit]=40") {
		assert.Fail(s.T(), "Limit is within range", "Expected limit to be size %d, but was %s", 40, *result.Links.First)
	}
	if !strings.Contains(*result.Links.First, "page[offset]=0") {
		assert.Fail(s.T(), "Offset is negative", "Expected offset to be %d, but was %s", 0, *result.Links.First)
	}
}

func (s *WorkItemSuite) TestPagingLinksHasAbsoluteURL() {
	// given
	offset := "10"
	limit := 10
	// when
	_, result := test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.HasPrefix(*result.Links.First, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "First", *result.Links.First)
	}
	if !strings.HasPrefix(*result.Links.Last, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Last", *result.Links.Last)
	}
	if !strings.HasPrefix(*result.Links.Prev, "http://") {
		assert.Fail(s.T(), "Not Absolute URL", "Expected link %s to contain absolute URL but was %s", "Prev", *result.Links.Prev)
	}
}

func (s *WorkItemSuite) TestPagingDefaultAndMaxSize() {
	// given
	offset := "0"
	var limit int
	// when
	_, result := test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, nil, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=20") {
		assert.Fail(s.T(), "Limit is nil", "Expected limit to be default size %d, got %v", 20, *result.Links.First)
	}
	// when
	limit = 1000
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=100") {
		assert.Fail(s.T(), "Limit is more than max", "Expected limit to be %d, got %v", 100, *result.Links.First)
	}
	// when
	limit = 50
	_, result = test.ListWorkitemsOK(s.T(), context.Background(), nil, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	if !strings.Contains(*result.Links.First, "page[limit]=50") {
		assert.Fail(s.T(), "Limit is within range", "Expected limit to be %d, got %v", 50, *result.Links.First)
	}
}

func (s *WorkItemSuite) TestGetWorkItemWithLegacyDescription() {
	// given
	_, wi := test.ShowWorkitemOK(s.T(), nil, nil, s.workitemCtrl, *s.wi.ID, nil, nil)
	require.NotNil(s.T(), wi)
	assert.Equal(s.T(), s.wi.ID, wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Attributes[workitem.SystemCreatedAt])
	assert.Equal(s.T(), s.testIdentity.ID.String(), *wi.Data.Relationships.Creator.Data.ID)
	// when
	wi.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	updatedDescription := "= Updated Test WI description"
	wi.Data.Attributes[workitem.SystemDescription] = updatedDescription
	payload2 := minimumRequiredUpdatePayload()
	payload2.Data.ID = wi.Data.ID
	payload2.Data.Attributes = wi.Data.Attributes
	_, updated := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &payload2)
	// then
	assert.NotNil(s.T(), updated.Data.Attributes[workitem.SystemCreatedAt])
	assert.Equal(s.T(), (s.wi.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(s.T(), *s.wi.ID, *updated.Data.ID)
	assert.Equal(s.T(), wi.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), updatedDescription, updated.Data.Attributes[workitem.SystemDescription])
}

func (s *WorkItemSuite) TestCreateWI() {
	// given
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when
	_, created := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	// then
	require.NotNil(s.T(), created.Data.ID)
	assert.NotEmpty(s.T(), *created.Data.ID)
	assert.NotNil(s.T(), created.Data.Attributes[workitem.SystemCreatedAt])
	assert.NotNil(s.T(), created.Data.Relationships.Creator.Data)
	assert.Equal(s.T(), *created.Data.Relationships.Creator.Data.ID, s.testIdentity.ID.String())
}

// TestReorderAbove is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result3 and places it **above** result2
func (s *WorkItemSuite) TestReorderWorkitemAboveOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	// This workitem is created but not used to clearly test that the reorder workitem is moved between **two** workitems i.e. result1 and result2 and not to the **top** of the list
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	_, result2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result3 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result3.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionAbove)
	_, reordered1 := test.ReorderWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result3.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result3.Data.ID, *reordered1.Data[0].ID)
}

// TestReorder is in error because of version conflict
func (s *WorkItemSuite) TestReorderWorkitemConflict() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	// This workitem is created but not used to clearly test that the reorder workitem is moved between **two** workitems i.e. result1 and result2 and not to the **top** of the list
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	_, result2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result3 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	result3.Data.Attributes["version"] = 101
	dataArray = append(dataArray, result3.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionAbove)

	_, err := test.ReorderWorkitemsConflict(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered

	require.NotNil(s.T(), err)
}

// TestReorderBelow is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result1 and places it **below** result1
func (s *WorkItemSuite) TestReorderWorkitemBelowOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	_, result1 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// This workitem is created but not used to clearly demonstrate that the reorder workitem is moved between **two** workitems i.e. result2 and result3 and not to the **bottom** of the list
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionBelow)

	_, reordered1 := test.ReorderWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result1.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result1.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderTop is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result2 and places it to the top of the list
func (s *WorkItemSuite) TestReorderWorkitemTopOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	// There are two workitems in the list -> result1 and result2
	// In this case, we reorder result2 to the top of the list i.e. above result1
	_, result1 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	payload2.Position.Direction = string(workitem.DirectionTop)
	_, reordered1 := test.ReorderWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result1.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result1.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderBottom is positive test which tests successful reorder by providing valid input
// This case reorders one workitem -> result1 and places it to the bottom of the list
func (s *WorkItemSuite) TestReorderWorkitemBottomOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed

	// There are two workitems in the list -> result1 and result2
	// In this case, we reorder result1 to the bottom of the list i.e. below result2
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, result2.Data)
	payload2.Data = dataArray
	payload2.Position.Direction = string(workitem.DirectionBottom)

	_, reordered1 := test.ReorderWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered

	require.Len(s.T(), reordered1.Data, 1) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result2.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), *result2.Data.ID, *reordered1.Data[0].ID)
}

// TestReorderMultipleWorkitem is positive test which tests successful reorder by providing valid input
// This case reorders two workitems -> result3 and result4 and places them above result2
func (s *WorkItemSuite) TestReorderMultipleWorkitems() {
	// given
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result3 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	_, result4 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()
	var dataArray []*app.WorkItem // dataArray contains the workitems that have to be reordered
	dataArray = append(dataArray, result3.Data, result4.Data)
	payload2.Data = dataArray
	payload2.Position.ID = result2.Data.ID // Position.ID specifies the workitem ID above or below which the workitem(s) should be placed
	payload2.Position.Direction = string(workitem.DirectionAbove)
	// when
	_, reordered1 := test.ReorderWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2) // Returns the workitems which are reordered
	// then
	require.NotNil(s.T(), reordered1)
	require.NotNil(s.T(), reordered1.Data)
	require.Len(s.T(), reordered1.Data, 2) // checks the correct number of workitems reordered
	assert.Equal(s.T(), result3.Data.Attributes["version"].(int)+1, reordered1.Data[0].Attributes["version"])
	assert.Equal(s.T(), result4.Data.Attributes["version"].(int)+1, reordered1.Data[1].Attributes["version"])
	assert.Equal(s.T(), *result3.Data.ID, *reordered1.Data[0].ID)
	assert.Equal(s.T(), *result4.Data.ID, *reordered1.Data[1].ID)
}

// TestReorderWorkitemBadRequest is negative test which tests unsuccessful reorder by providing invalid input
func (s *WorkItemSuite) TestReorderWorkitemBadRequestOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	_, result1 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	// This case gives empty dataArray as input
	// Response is Bad Parameter
	// Reorder is unsuccessful

	var dataArray []*app.WorkItem
	payload2.Data = dataArray
	payload2.Position.ID = result1.Data.ID
	payload2.Position.Direction = string(workitem.DirectionAbove)
	test.ReorderWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2)
}

// TestReorderWorkitemNotFound is negative test which tests unsuccessful reorder by providing invalid input
func (s *WorkItemSuite) TestReorderWorkitemNotFoundOK() {
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Reorder Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	_, result1 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	payload2 := minimumRequiredReorderPayload()

	// This case gives id of workitem in position.ID which is not present in db as input
	// Response is Not Found
	// Reorder is unsuccessful

	var dataArray []*app.WorkItem
	dataArray = append(dataArray, result1.Data)
	payload2.Data = dataArray
	randomID := uuid.NewV4()
	payload2.Position.ID = &randomID
	payload2.Position.Direction = string(workitem.DirectionAbove)
	test.ReorderWorkitemsNotFound(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, &payload2)
}

// TestUpdateWorkitemWithoutReorder tests that when workitem is updated, execution order of workitem doesnot change.
func (s *WorkItemSuite) TestUpdateWorkitemWithoutReorder() {

	// Create new workitem
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// Update the workitem
	wi.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	payload2 := minimumRequiredUpdatePayload()
	payload2.Data.ID = wi.Data.ID
	payload2.Data.Attributes = wi.Data.Attributes
	_, updated := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &payload2)

	assert.Equal(s.T(), *wi.Data.ID, *updated.Data.ID)
	assert.Equal(s.T(), (s.wi.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(s.T(), wi.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])

	// Check the execution order
	assert.Equal(s.T(), wi.Data.Attributes[workitem.SystemOrder], updated.Data.Attributes[workitem.SystemOrder])
}

func (s *WorkItemSuite) TestCreateWorkItemWithoutContext() {
	// given
	s.svc = goa.New("TestCreateWorkItemWithoutContext-Service")
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when/then
	test.CreateWorkitemsUnauthorized(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
}

func (s *WorkItemSuite) TestListByFields() {
	// given
	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "run integration test"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateClosed
	test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	// when
	filter := "{\"system.title\":\"run integration test\"}"
	offset := "0"
	limit := 1
	_, result := test.ListWorkitemsOK(s.T(), nil, nil, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &filter, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	// then
	require.NotNil(s.T(), result)
	require.Equal(s.T(), 1, len(result.Data))
	// when
	filter = fmt.Sprintf("{\"system.creator\":\"%s\"}", s.testIdentity.ID.String())
	// then
	_, result = test.ListWorkitemsOK(s.T(), nil, nil, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &filter, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
	require.NotNil(s.T(), result)
	require.Equal(s.T(), 1, len(result.Data))
}
func getWorkItemTestDataFunc(config configuration.ConfigurationData) func(t *testing.T) []testSecureAPI {
	return func(t *testing.T) []testSecureAPI {
		privatekey, err := jwt.ParseRSAPrivateKeyFromPEM(config.GetTokenPrivateKey())
		if err != nil {
			t.Fatal("Could not parse Key ", err)
		}
		differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))

		if err != nil {
			t.Fatal("Could not parse different private key ", err)
		}

		createWIPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"type": "workitems"
				"attributes": {
					"system.state": "new",
					"system.title": "My special story",
					"system.description": "description"
				}
			}
		}`))

		return []testSecureAPI{
			// Create Work Item API with different parameters
			{
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItems, "5b5faa94-7478-4a35-9fdd-e1b5278df331"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItems, "5b5faa94-7478-4a35-9fdd-e1b5278df331"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItems, "5b5faa94-7478-4a35-9fdd-e1b5278df331"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPost,
				url:                fmt.Sprintf(endpointWorkItems, "5b5faa94-7478-4a35-9fdd-e1b5278df331"),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           "",
			},
			// Update Work Item API with different parameters
			{
				method:             http.MethodPatch,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodPatch,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            createWIPayloadString,
				jwtToken:           "",
			},
			// Delete Work Item API with different parameters
			{
				method:             http.MethodDelete,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getExpiredAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getMalformedAuthHeader(t, privatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           getValidAuthHeader(t, differentPrivatekey),
			}, {
				method:             http.MethodDelete,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusUnauthorized,
				expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
				payload:            nil,
				jwtToken:           "",
			},
			// Try fetching a random work Item
			// We do not have security on GET hence this should return 404 not found
			{
				method:             http.MethodGet,
				url:                endpointWorkItem + "/" + uuid.NewV4().String(),
				expectedStatusCode: http.StatusNotFound,
				expectedErrorCode:  jsonapi.ErrorCodeNotFound,
				payload:            nil,
				jwtToken:           "",
			},
		}
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *WorkItemSuite) TestUnauthorizeWorkItemCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemTestDataFunc(*s.Configuration), func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWI-Service")
	}, func(service *goa.Service) error {
		workitemCtrl := NewWorkitemController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkitemController(service, workitemCtrl)
		workitemsCtrl := NewWorkitemsController(service, gormapplication.NewGormDB(s.DB), s.Configuration)
		app.MountWorkitemsController(service, workitemsCtrl)
		return nil
	})
}

func createPagingTest(t *testing.T, ctx context.Context, controller *WorkitemsController, repo *workitem.WorkItemRepository, spaceID uuid.UUID, totalCount int) func(start int, limit int, first string, last string, prev string, next string) {
	return func(start int, limit int, first string, last string, prev string, next string) {
		offset := strconv.Itoa(start)

		_, response := test.ListWorkitemsOK(t, ctx, nil, controller, spaceID, nil, nil, nil, nil, nil, nil, nil, nil, &limit, &offset, nil, nil)
		assertLink(t, "first", first, response.Links.First)
		assertLink(t, "last", last, response.Links.Last)
		assertLink(t, "prev", prev, response.Links.Prev)
		assertLink(t, "next", next, response.Links.Next)
		assert.Equal(t, totalCount, response.Meta.TotalCount)
	}
}

func assertLink(t *testing.T, l string, expected string, actual *string) {
	if expected == "" {
		if actual != nil {
			assert.Fail(t, fmt.Sprintf("link %s should be nil but is %s", l, *actual))
		}
	} else {
		if actual == nil {
			assert.Fail(t, "link %s should be %s, but is nil", l, expected)
		} else {
			assert.True(t, strings.HasSuffix(*actual, expected), "link %s should be %s, but is %s", l, expected, *actual)
		}
	}
}

// ========== helper functions for tests inside WorkItem2Suite ==========
func getMinimumRequiredUpdatePayload(wi *app.WorkItem) *app.UpdateWorkitemPayload {
	return &app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			Type: APIStringTypeWorkItem,
			ID:   wi.ID,
			Attributes: map[string]interface{}{
				"version": wi.Attributes["version"],
			},
			Relationships: wi.Relationships,
		},
	}
}

func minimumRequiredUpdatePayload() app.UpdateWorkitemPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	return app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
}

func minimumRequiredReorderPayload() app.ReorderWorkitemsPayload {
	return app.ReorderWorkitemsPayload{
		Data: []*app.WorkItem{},
		Position: &app.WorkItemReorderPosition{
			ID: nil,
		},
	}
}

func minimumRequiredCreateWithType(witID uuid.UUID) app.CreateWorkitemsPayload {
	c := minimumRequiredCreatePayload()
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, witID)
	return c
}

func minimumRequiredCreateWithTypeAndSpace(witID uuid.UUID, spaceID uuid.UUID) app.CreateWorkitemsPayload {
	c := minimumRequiredCreatePayload()
	c.Data.Relationships.BaseType = newRelationBaseType(spaceID, witID)
	return c
}

func newRelationBaseType(spaceID, wit uuid.UUID) *app.RelationBaseType {
	witRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(spaceID.String(), wit.String()))

	return &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   wit,
		},
		Links: &app.GenericLinks{
			Self:    &witRelatedURL,
			Related: &witRelatedURL,
		},
	}
}

func minimumRequiredCreatePayload() app.CreateWorkitemsPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))

	return app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
}

func createOneRandomUserIdentity(ctx context.Context, db *gorm.DB) *account.Identity {
	newUserUUID := uuid.NewV4()
	identityRepo := account.NewIdentityRepository(db)
	identity := account.Identity{
		Username: "Test User Integration Random",
		ID:       newUserUUID,
	}
	err := identityRepo.Create(ctx, &identity)
	if err != nil {
		fmt.Println("should not happen off.")
		return nil
	}
	return &identity
}

func createOneRandomIteration(ctx context.Context, db *gorm.DB) *iteration.Iteration {
	iterationRepo := iteration.NewIterationRepository(db)
	spaceRepo := space.NewRepository(db)

	// added timestmap to the space in order to make this function usable for more than one test
	// else it fails with - (pq: duplicate key value violates unique constraint "spaces_name_idx")
	newSpace := space.Space{
		Name: "Space iteration " + time.Now().String(),
	}
	space, err := spaceRepo.Create(ctx, &newSpace)
	if err != nil {
		fmt.Println("Failed to create space for iteration.")
		return nil
	}

	itr := iteration.Iteration{
		Name:    "Sprint 101",
		SpaceID: space.ID,
	}
	err = iterationRepo.Create(ctx, &itr)
	if err != nil {
		fmt.Println("Failed to create iteration.")
		return nil
	}
	return &itr
}

func createOneRandomArea(ctx context.Context, db *gorm.DB, testName string) *area.Area {
	areaRepo := area.NewAreaRepository(db)
	spaceRepo := space.NewRepository(db)

	newSpace := space.Space{
		Name: fmt.Sprintf("Space area %v %v", testName, uuid.NewV4()),
	}
	space, err := spaceRepo.Create(ctx, &newSpace)
	if err != nil {
		fmt.Println("Failed to create space for area.")
		return nil
	}
	ar := area.Area{
		Name:    "Area 51",
		SpaceID: space.ID,
	}
	err = areaRepo.Create(ctx, &ar)
	if err != nil {
		fmt.Println("Failed to create area.")
		return nil
	}
	return &ar
}

func newChildIteration(ctx context.Context, db *gorm.DB, parentIteration *iteration.Iteration) *iteration.Iteration {
	iterationRepo := iteration.NewIterationRepository(db)

	parentPath := append(parentIteration.Path, parentIteration.ID)
	itr := iteration.Iteration{
		Name:    "Sprint 101",
		SpaceID: parentIteration.SpaceID,
		Path:    parentPath,
	}
	err := iterationRepo.Create(ctx, &itr)
	if err != nil {
		fmt.Println("Failed to create iteration.")
		return nil
	}
	return &itr
}

// ========== WorkItem2Suite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
// a normal test function that will kick off WorkItem2Suite
func TestSuiteWorkItem2(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &WorkItem2Suite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func ident(id uuid.UUID) *app.GenericData {
	ut := APIStringTypeUser
	i := id.String()
	return &app.GenericData{
		Type: &ut,
		ID:   &i,
	}
}

type WorkItem2Suite struct {
	gormtestsupport.DBTestSuite
	clean          func()
	workitemCtrl   app.WorkitemController
	workitemsCtrl  app.WorkitemsController
	linkCtrl       app.WorkItemLinkController
	linkCatCtrl    app.WorkItemLinkCategoryController
	linkTypeCtrl   app.WorkItemLinkTypeController
	spaceCtrl      app.SpaceController
	pubKey         *rsa.PublicKey
	priKey         *rsa.PrivateKey
	svc            *goa.Service
	wi             *app.WorkItem
	minimumPayload *app.UpdateWorkitemPayload
	ctx            context.Context
	notification   testsupport.NotificationChannel
}

func (s *WorkItem2Suite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *WorkItem2Suite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	s.notification = testsupport.NotificationChannel{}
	// create identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "WorkItem2Suite setup user", "test provider")
	require.Nil(s.T(), err)
	s.priKey, _ = wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("TestUpdateWI2-Service", wittoken.NewManagerWithPrivateKey(s.priKey), *testIdentity)
	s.workitemCtrl = NewNotifyingWorkitemController(s.svc, gormapplication.NewGormDB(s.DB), &s.notification, s.Configuration)
	s.workitemsCtrl = NewNotifyingWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), &s.notification, s.Configuration)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.DB))
	s.linkTypeCtrl = NewWorkItemLinkTypeController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.linkCtrl = NewWorkItemLinkController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.spaceCtrl = NewSpaceController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration, &DummyResourceManager{})

	payload := minimumRequiredCreateWithType(workitem.SystemBug)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	s.wi = wi.Data
	s.minimumPayload = getMinimumRequiredUpdatePayload(s.wi)
	//s.minimumReorderPayload = getMinimumRequiredReorderPayload(s.wi)
}

func (s *WorkItem2Suite) TearDownTest() {
	s.clean()
}

// ========== Actual Test functions ==========
func (s *WorkItem2Suite) TestWI2UpdateOnlyState() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	s.minimumPayload.Data.Attributes["system.state"] = "invalid_value"
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	newStateValue := "closed"
	s.minimumPayload.Data.Attributes[workitem.SystemState] = newStateValue
	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), updatedWI.Data.Attributes[workitem.SystemState], newStateValue)
}

func (s *WorkItem2Suite) TestWI2UpdateVersionConflict() {
	// given
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	s.minimumPayload.Data.Attributes["version"] = 2398475203
	// when/then
	test.UpdateWorkitemConflict(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
}

func (s *WorkItem2Suite) TestWI2UpdateWithNonExistentID() {
	id := uuid.NewV4()
	s.minimumPayload.Data.ID = &id
	test.UpdateWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.workitemCtrl, id, s.minimumPayload)
}

func (s *WorkItem2Suite) TestWI2UpdateSetBaseType() {
	c := minimumRequiredCreateWithType(workitem.SystemBug)
	c.Data.Attributes[workitem.SystemTitle] = "Test title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	_, created := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	assert.Equal(s.T(), created.Data.Relationships.BaseType.Data.ID, workitem.SystemBug)

	u := minimumRequiredUpdatePayload()
	u.Data.Attributes[workitem.SystemTitle] = "Test title"
	u.Data.Attributes["version"] = created.Data.Attributes["version"]
	u.Data.ID = created.Data.ID
	u.Data.Relationships = &app.WorkItemRelationships{
		BaseType: newRelationBaseType(space.SystemSpace, workitem.SystemExperience),
	}

	_, newWi := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, &u)

	// Ensure the type wasn't updated
	require.Equal(s.T(), workitem.SystemBug, newWi.Data.Relationships.BaseType.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyLegacyDescription() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := "Only Description is modified"
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "Only Description is modified"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// fixing https://github.com/fabric8-services/fabric8-wit/issues/986
func (s *WorkItem2Suite) TestWI2UpdateDescriptionAndMarkup() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := "# Description is modified"
	expectedDescription := "# Description is modified"
	expectedRenderedDescription := "<h1>Description is modified</h1>\n"
	modifiedMarkup := rendering.SystemMarkupMarkdown
	expectedMarkup := rendering.SystemMarkupMarkdown
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription
	s.minimumPayload.Data.Attributes[workitem.SystemDescriptionMarkup] = modifiedMarkup

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), expectedMarkup, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyMarkupDescriptionWithoutMarkup() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := rendering.NewMarkupContentFromLegacy("Only Description is modified")
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "Only Description is modified"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription.ToMap()
	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateOnlyMarkupDescriptionWithMarkup() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	modifiedDescription := rendering.NewMarkupContent("Only Description is modified", rendering.SystemMarkupMarkdown)
	expectedDescription := "Only Description is modified"
	expectedRenderedDescription := "<p>Only Description is modified</p>\n"
	s.minimumPayload.Data.Attributes[workitem.SystemDescription] = modifiedDescription.ToMap()

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), expectedDescription, updatedWI.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), expectedRenderedDescription, updatedWI.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, updatedWI.Data.Attributes[workitem.SystemDescriptionMarkup])
}

func (s *WorkItem2Suite) TestWI2UpdateMultipleScenarios() {
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	// update title attribute
	modifiedTitle := "Is the model updated?"
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = modifiedTitle

	_, updatedWI := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), updatedWI.Data.Attributes[workitem.SystemTitle], modifiedTitle)

	// verify self link value
	if !strings.HasPrefix(updatedWI.Links.Self, "http://") {
		assert.Fail(s.T(), fmt.Sprintf("%s is not absolute URL", updatedWI.Links.Self))
	}
	if !strings.HasSuffix(updatedWI.Links.Self, fmt.Sprintf("/%s", *updatedWI.Data.ID)) {
		assert.Fail(s.T(), fmt.Sprintf("%s is not FETCH URL of the resource", updatedWI.Links.Self))
	}
	// clean up and keep version updated in order to keep object future usage
	delete(s.minimumPayload.Data.Attributes, workitem.SystemTitle)
	s.minimumPayload.Data.Attributes[workitem.SystemTitle] = "Test title"
	s.minimumPayload.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]

	// update assignee relationship and verify
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	require.NotNil(s.T(), newUser)

	newUserUUID := newUser.ID.String()
	s.minimumPayload.Data.Relationships = &app.WorkItemRelationships{}

	userType := APIStringTypeUser
	// update with invalid assignee string (non-UUID)
	maliciousUUID := "non UUID string"
	s.minimumPayload.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &maliciousUUID,
				Type: &userType,
			}},
	}
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)

	s.minimumPayload.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				ID:   &newUserUUID,
				Type: &userType,
			}},
	}

	_, updatedWI = test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	assert.Equal(s.T(), *updatedWI.Data.Relationships.Assignees.Data[0].ID, newUser.ID.String())

	// update to wrong version
	correctVersion := updatedWI.Data.Attributes["version"]
	s.minimumPayload.Data.Attributes["version"] = 12453972348
	test.UpdateWorkitemConflict(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	s.minimumPayload.Data.Attributes["version"] = correctVersion

	// Add test to remove assignee for WI
	s.minimumPayload.Data.Relationships.Assignees.Data = nil
	_, updatedWI = test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *s.wi.ID, s.minimumPayload)
	require.NotNil(s.T(), updatedWI)
	require.Len(s.T(), updatedWI.Data.Relationships.Assignees.Data, 0)
	// need to do in order to keep object future usage
	s.minimumPayload.Data.Attributes["version"] = updatedWI.Data.Attributes["version"]
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItem() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), "Title", wi.Data.Attributes[workitem.SystemTitle])
	assert.NotNil(s.T(), wi.Data.Attributes[workitem.SystemNumber])
	assert.NotNil(s.T(), wi.Data.Relationships.BaseType.Data.ID)
	assert.NotNil(s.T(), wi.Data.Relationships.Comments.Links.Self)
	assert.NotNil(s.T(), wi.Data.Relationships.Area.Data.ID)
	assert.NotNil(s.T(), wi.Data.Relationships.Creator.Data.ID)
	assert.NotNil(s.T(), wi.Data.Links)
	assert.NotNil(s.T(), wi.Data.Links.Self)
}

// TestWI2SuccessCreateWorkItemWithoutDescription verifies that the `workitem.SystemDescription` attribute is not set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is not provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithoutDescription() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescription])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescriptionMarkup])
	assert.Nil(s.T(), wi.Data.Attributes[workitem.SystemDescriptionRendered])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithLegacyDescription() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = "Description"
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithDescriptionAndMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContent("Description", rendering.SystemMarkupMarkdown)
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "<p>Description</p>\n", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set as a MarkupContent element
func (s *WorkItem2Suite) TestWI2SuccessCreateWorkItemWithDescriptionAndNoMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContentFromLegacy("Description")
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.Attributes)
	assert.Equal(s.T(), c.Data.Attributes[workitem.SystemTitle], wi.Data.Attributes[workitem.SystemTitle])
	// for now, we keep the legacy format in the output
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescription])
	assert.Equal(s.T(), "Description", wi.Data.Attributes[workitem.SystemDescriptionRendered])
	assert.Equal(s.T(), rendering.SystemMarkupDefault, wi.Data.Attributes[workitem.SystemDescriptionMarkup])
}

// TestWI2SuccessCreateWorkItemWithDescription verifies that the `workitem.SystemDescription` attribute is set, as well as its pair workitem.SystemDescriptionMarkup when the work item description is provided
func (s *WorkItem2Suite) TestWI2FailCreateWorkItemWithDescriptionAndUnsupportedMarkup() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemDescription] = rendering.NewMarkupContent("Description", "foo")
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateMissingBaseType() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithAssigneeAsField() {
	// given
	s.T().Skip("Not working.. require WIT understanding on server side")
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Attributes[workitem.SystemAssignees] = []string{"34343"}
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Nil(s.T(), wi.Data.Relationships.Assignees.Data)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithMissingTitle() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2FailCreateWithEmptyTitle() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = ""
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWithAssigneeRelation() {
	// given
	userType := APIStringTypeUser
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	newUserID := newUser.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				Type: &userType,
				ID:   &newUserID,
			}},
	}
	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.NotNil(s.T(), wi.Data.Relationships.Assignees.Data)
	assert.NotNil(s.T(), wi.Data.Relationships.Assignees.Data[0].ID)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateWithAssigneesRelation() {
	// given
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	newUser2 := createOneRandomUserIdentity(s.svc.Context, s.DB)
	newUser3 := createOneRandomUserIdentity(s.svc.Context, s.DB)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
		},
	}
	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *wi.Data.Relationships.Assignees.Data[0].ID)
	update := minimumRequiredUpdatePayload()
	update.Data.ID = wi.Data.ID
	update.Data.Type = wi.Data.Type
	update.Data.Attributes[workitem.SystemTitle] = "Title"
	update.Data.Attributes["version"] = wi.Data.Attributes["version"]
	spaceRelation := update.Data.Relationships.Space
	update.Data.Relationships = &app.WorkItemRelationships{
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(newUser2.ID),
				ident(newUser3.ID),
			},
		},
		Space: spaceRelation,
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &update)
	assert.Len(s.T(), wiu.Data.Relationships.Assignees.Data, 2)
	assert.Equal(s.T(), newUser2.ID.String(), *wiu.Data.Relationships.Assignees.Data[0].ID)
	assert.Equal(s.T(), newUser3.ID.String(), *wiu.Data.Relationships.Assignees.Data[1].ID)
}

func (s *WorkItem2Suite) TestWI2ListByAssigneeFilter() {
	// given
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
		},
	}
	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *wi.Data.Relationships.Assignees.Data[0].ID)
	newUserID := newUser.ID.String()
	_, list := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, &newUserID, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), newUser.ID.String(), *list.Data[0].Relationships.Assignees.Data[0].ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[assignee]"))
}

func (s *WorkItem2Suite) TestWI2ListByNoAssigneeFilter() {
	// given
	userType := APIStringTypeUser
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	newUserID := newUser.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			{
				Type: &userType,
				ID:   &newUserID,
			}},
	}
	assignee := none

	s.T().Run("default work item created in fixture", func(t *testing.T) {
		_, list0 := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, &assignee, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		// data coming from test fixture
		assert.Len(t, list0.Data, 1)
		assert.True(t, strings.Contains(*list0.Links.First, "filter[assignee]=none"))
	})

	s.T().Run("work item with assignee", func(t *testing.T) {
		_, wi := test.CreateWorkitemsCreated(t, s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
		assert.NotNil(t, wi.Data)
		assert.NotNil(t, wi.Data.ID)
		assert.NotNil(t, wi.Data.Type)
		assert.NotNil(t, wi.Data.Attributes)
		assert.NotNil(t, wi.Data.Relationships.Assignees.Data)
		assert.NotNil(t, wi.Data.Relationships.Assignees.Data[0].ID)

		_, list := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, &newUserID, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		assert.Len(t, list.Data, 1)
		require.NotNil(t, *list.Data[0].Relationships.Assignees.Data[0])
		assert.Equal(t, newUser.ID.String(), *list.Data[0].Relationships.Assignees.Data[0].ID)
		assert.False(t, strings.Contains(*list.Links.First, "filter[assignee]=none"))
	})

	s.T().Run("work item with assignee value as none", func(t *testing.T) {
		_, list2 := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, &assignee, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		assert.Len(t, list2.Data, 1)
		assert.True(t, strings.Contains(*list2.Links.First, "filter[assignee]=none"))
	})

	s.T().Run("work item without specifying assignee", func(t *testing.T) {
		_, list3 := test.ListWorkitemsOK(t, s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		assert.Len(t, list3.Data, 2)
		assert.False(t, strings.Contains(*list3.Links.First, "filter[assignee]=none"))
	})
}

func (s *WorkItem2Suite) TestWI2ListByTypeFilter() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	// when
	_, expected := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	assert.NotNil(s.T(), expected.Data)
	require.NotNil(s.T(), expected.Data.ID)
	require.NotNil(s.T(), expected.Data.Type)
	_, actual := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, nil, &workitem.SystemBug, nil, nil, nil, nil)
	require.NotNil(s.T(), actual)
	require.True(s.T(), len(actual.Data) > 1)
	assert.Contains(s.T(), *actual.Links.First, fmt.Sprintf("filter[workitemtype]=%s", workitem.SystemBug))
	for _, actualWI := range actual.Data {
		assert.Equal(s.T(), expected.Data.Type, actualWI.Type)
		require.NotNil(s.T(), actualWI.ID)
	}
}

func (s *WorkItem2Suite) createWorkItem(title, state string) app.WorkItemSingle {
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = state
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	return *wi
}

func (s *WorkItem2Suite) TestWI2ListByStateFilterOK() {
	// given
	_ = s.createWorkItem("title", workitem.SystemStateNew)
	inprogressWI := s.createWorkItem("title", workitem.SystemStateInProgress)
	// when
	stateNew := workitem.SystemStateNew
	_, actualWIs := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, &stateNew, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), actualWIs)
	require.True(s.T(), len(actualWIs.Data) > 1)
	assert.Contains(s.T(), *actualWIs.Links.First, fmt.Sprintf("filter[workitemstate]=%s", workitem.SystemStateNew))
	for _, actualWI := range actualWIs.Data {
		require.NotNil(s.T(), actualWI.Attributes[workitem.SystemState])
		assert.Equal(s.T(), stateNew, actualWI.Attributes[workitem.SystemState])
		assert.NotEqual(s.T(), *inprogressWI.Data.ID, *actualWI.ID)
	}
}

// see https://github.com/fabric8-services/fabric8-wit/issues/1268
func (s *WorkItem2Suite) TestWI2ListByStateFilterNotModifiedUsingIfNoneMatchIfModifiedSinceHeaders() {
	// given
	_ = s.createWorkItem("title", workitem.SystemStateNew)
	inprogressWI := s.createWorkItem("title", workitem.SystemStateInProgress)
	// when
	stateNew := workitem.SystemStateNew
	res, actualWIs := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, &stateNew, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), actualWIs)
	require.True(s.T(), len(actualWIs.Data) > 1)
	assert.Contains(s.T(), *actualWIs.Links.First, fmt.Sprintf("filter[workitemstate]=%s", workitem.SystemStateNew))
	for _, actualWI := range actualWIs.Data {
		require.NotNil(s.T(), actualWI.Attributes[workitem.SystemState])
		assert.Equal(s.T(), stateNew, actualWI.Attributes[workitem.SystemState])
		assert.NotEqual(s.T(), *inprogressWI.Data.ID, *actualWI.ID)
	}
	// retain conditional headers in response and submit the request again
	etag, lastModified, _ := assertResponseHeaders(s.T(), res)
	// when calling again
	res = test.ListWorkitemsNotModified(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, &stateNew, nil, nil, nil, &lastModified, &etag)
	// then
	assertResponseHeaders(s.T(), res)
}

// see https://github.com/fabric8-services/fabric8-wit/issues/1268
func (s *WorkItem2Suite) TestWI2ListByStateFilterOKModifiedUsingIfNoneMatchIfModifiedSinceHeaders() {
	// given
	_ = s.createWorkItem("title", workitem.SystemStateNew)
	inprogressWI := s.createWorkItem("title", workitem.SystemStateInProgress)
	// when
	stateNew := workitem.SystemStateNew
	res, actualWIs := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, &stateNew, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), actualWIs)
	require.True(s.T(), len(actualWIs.Data) > 1)
	assert.Contains(s.T(), *actualWIs.Links.First, fmt.Sprintf("filter[workitemstate]=%s", workitem.SystemStateNew))
	for _, actualWI := range actualWIs.Data {
		require.NotNil(s.T(), actualWI.Attributes[workitem.SystemState])
		assert.Equal(s.T(), stateNew, actualWI.Attributes[workitem.SystemState])
		assert.NotEqual(s.T(), *inprogressWI.Data.ID, *actualWI.ID)
	}
	// retain conditional headers in response and submit the request again
	etag, lastModified, _ := assertResponseHeaders(s.T(), res)
	// modify the state of the inprogressWI
	update := minimumRequiredUpdatePayload()
	update.Data.ID = inprogressWI.Data.ID
	update.Data.Type = inprogressWI.Data.Type
	update.Data.Attributes[workitem.SystemTitle] = inprogressWI.Data.Attributes[workitem.SystemTitle]
	update.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	update.Data.Attributes["version"] = inprogressWI.Data.Attributes["version"]
	test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *inprogressWI.Data.ID, &update)
	// when calling again (with expired validation headers)
	res, actualWIs = test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, nil, nil, &stateNew, nil, nil, nil, &lastModified, &etag)
	// then expect the new data
	assertResponseHeaders(s.T(), res)
	require.NotNil(s.T(), actualWIs)
	require.True(s.T(), len(actualWIs.Data) > 2)
	assert.Contains(s.T(), *actualWIs.Links.First, fmt.Sprintf("filter[workitemstate]=%s", workitem.SystemStateNew))
	for _, actualWI := range actualWIs.Data {
		require.NotNil(s.T(), actualWI.Attributes[workitem.SystemState])
		assert.Equal(s.T(), stateNew, actualWI.Attributes[workitem.SystemState])
	}
}

func (s *WorkItem2Suite) setupAreaWorkItem(createWorkItem bool) (uuid.UUID, string, *app.WorkItemSingle) {
	tempArea := createOneRandomArea(s.svc.Context, s.DB, "TestWI2ListByAreaFilter")
	require.NotNil(s.T(), tempArea)
	areaID := tempArea.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Area = &app.RelationGeneric{
		Data: &app.GenericData{
			ID: &areaID,
		},
	}
	if createWorkItem {
		_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
		require.NotNil(s.T(), wi)
		require.NotNil(s.T(), wi.Data)
		require.NotNil(s.T(), wi.Data.ID)
		require.NotNil(s.T(), wi.Data.Type)
		require.NotNil(s.T(), wi.Data.Attributes)
		require.NotNil(s.T(), wi.Data.Relationships.Area)
		assert.Equal(s.T(), areaID, *wi.Data.Relationships.Area.Data.ID)
		return *c.Data.Relationships.Space.Data.ID, areaID, wi
	}
	return *c.Data.Relationships.Space.Data.ID, areaID, nil
}

func assertAreaWorkItems(t *testing.T, areaID string, workitems *app.WorkItemList) {
	require.NotNil(t, workitems)
	require.NotNil(t, workitems.Data)
	require.Len(t, workitems.Data, 1)
	assert.Equal(t, areaID, *workitems.Data[0].Relationships.Area.Data.ID)
	assert.True(t, strings.Contains(*workitems.Links.First, "filter[area]"))
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterOK() {
	// given
	spaceID, areaID, _ := s.setupAreaWorkItem(true)
	// when
	res, workitems := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	assertAreaWorkItems(s.T(), areaID, workitems)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterOKEmptyList() {
	// given
	spaceID, areaID, _ := s.setupAreaWorkItem(false)
	// when
	res, workitems := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	// then
	require.NotNil(s.T(), *workitems)
	require.Empty(s.T(), workitems.Data)
	// should not be the default/nil time
	var defaultTime time.Time
	assert.NotEqual(s.T(), app.ToHTTPTime(defaultTime), res.Header().Get(app.LastModified))
	assert.Equal(s.T(), app.GenerateEmptyTag(), res.Header().Get(app.ETag))
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterOKUsingExpiredIfModifiedSinceHeader() {
	// given
	spaceID, areaID, wi := s.setupAreaWorkItem(true)
	// when
	updatedAt := wi.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt.Add(-1 * time.Hour))
	res, workitems := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	assertAreaWorkItems(s.T(), areaID, workitems)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterOKUsingExpiredIfNoneMatchHeader() {
	// given
	spaceID, areaID, _ := s.setupAreaWorkItem(true)
	// when
	ifNoneMatch := "foo"
	res, workitems := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	assertAreaWorkItems(s.T(), areaID, workitems)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterNotModifiedUsingIfModifiedSinceHeader() {
	// given
	spaceID, areaID, wi := s.setupAreaWorkItem(true)
	// when
	updatedAt := wi.Data.Attributes[workitem.SystemUpdatedAt].(time.Time)
	ifModifiedSince := app.ToHTTPTime(updatedAt)
	res := test.ListWorkitemsNotModified(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ListByAreaFilterNotModifiedUsingIfNoneMatchHeader() {
	// given
	spaceID, areaID, wi := s.setupAreaWorkItem(true)
	// when
	ifNoneMatch := app.GenerateEntityTag(convertWorkItemToConditionalRequestEntity(*wi))
	res := test.ListWorkitemsNotModified(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, nil, &areaID, nil, nil, nil, nil, nil, nil, nil, nil, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ListByIterationFilter() {
	tempIteration := createOneRandomIteration(s.svc.Context, s.DB)
	require.NotNil(s.T(), tempIteration)
	iterationID := tempIteration.ID.String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			ID: &iterationID,
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), wi.Data)
	require.NotNil(s.T(), wi.Data.ID)
	require.NotNil(s.T(), wi.Data.Type)
	require.NotNil(s.T(), wi.Data.Attributes)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	assert.Equal(s.T(), iterationID, *wi.Data.Relationships.Iteration.Data.ID)

	_, list := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, nil, nil, &iterationID, nil, nil, nil, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 1)
	assert.Equal(s.T(), iterationID, *list.Data[0].Relationships.Iteration.Data.ID)
	assert.True(s.T(), strings.Contains(*list.Links.First, "filter[iteration]"))
}

func (s *WorkItem2Suite) TestWI2FailCreateInvalidAssignees() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(uuid.NewV4()),
		},
	}
	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2FailUpdateInvalidAssignees() {
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)

	update := minimumRequiredUpdatePayload()
	update.Data.ID = wi.Data.ID
	update.Data.Type = wi.Data.Type
	update.Data.Attributes["version"] = wi.Data.Attributes["version"]
	update.Data.Relationships = &app.WorkItemRelationships{
		Assignees: &app.RelationGenericList{
			Data: []*app.GenericData{
				ident(uuid.NewV4()),
			},
		},
	}
	test.UpdateWorkitemBadRequest(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &update)
}

func (s *WorkItem2Suite) TestWI2SuccessUpdateWithAssigneesRelation() {
	newUser := createOneRandomUserIdentity(s.svc.Context, s.DB)
	newUser2 := createOneRandomUserIdentity(s.svc.Context, s.DB)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Assignees = &app.RelationGenericList{
		Data: []*app.GenericData{
			ident(newUser.ID),
			ident(newUser2.ID),
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	assert.NotNil(s.T(), wi.Data)
	assert.NotNil(s.T(), wi.Data.ID)
	assert.NotNil(s.T(), wi.Data.Type)
	assert.NotNil(s.T(), wi.Data.Attributes)
	assert.Len(s.T(), wi.Data.Relationships.Assignees.Data, 2)
}

func (s *WorkItem2Suite) TestWI2ShowOK() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	res, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	// then
	assertSingleWorkItem(s.T(), *createdWI, *fetchedWI)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ShowOKUsingExpiredIfModifiedSinceHeader() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	ifModifiedSince := app.ToHTTPTime(createdWI.Data.Attributes[workitem.SystemUpdatedAt].(time.Time).Add(-10 * time.Hour))
	res, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, &ifModifiedSince, nil)
	// then
	assertSingleWorkItem(s.T(), *createdWI, *fetchedWI)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ShowOKUsingExpiredIfNoneMatchHeader() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	ifNoneMatch := "foo"
	res, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, &ifNoneMatch)
	// then
	assertSingleWorkItem(s.T(), *createdWI, *fetchedWI)
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ShowNotModifiedUsingIfModifiedSinceHeader() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	ifModifiedSince := app.ToHTTPTime(createdWI.Data.Attributes[workitem.SystemUpdatedAt].(time.Time))
	res := test.ShowWorkitemNotModified(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *WorkItem2Suite) TestWI2ShowNotModifiedUsingIfNoneMatchHeader() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	ifNoneMatch := app.GenerateEntityTag(convertWorkItemToConditionalRequestEntity(*createdWI))
	res := test.ShowWorkitemNotModified(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func assertSingleWorkItem(t *testing.T, createdWI app.WorkItemSingle, fetchedWI app.WorkItemSingle) {
	assert.NotNil(t, fetchedWI.Data)
	assert.NotNil(t, fetchedWI.Data.ID)
	assert.Equal(t, createdWI.Data.ID, fetchedWI.Data.ID)
	assert.NotNil(t, fetchedWI.Data.Type)
	assert.NotNil(t, fetchedWI.Data.Attributes)
	assert.NotNil(t, fetchedWI.Data.Links.Self)
	assert.NotNil(t, fetchedWI.Data.Relationships.Creator.Data.ID)
	assert.NotNil(t, fetchedWI.Data.Relationships.BaseType.Data.ID)
}

func assertResponseHeaders(t *testing.T, res http.ResponseWriter) (string, string, string) {
	lastModified := res.Header()[app.LastModified]
	etag := res.Header()[app.ETag]
	cacheControl := res.Header()[app.CacheControl]
	assert.NotEmpty(t, lastModified)
	assert.NotEmpty(t, etag)
	assert.NotEmpty(t, cacheControl)
	return etag[0], lastModified[0], cacheControl[0]
}

func (s *WorkItem2Suite) TestWI2FailShowMissing() {
	test.ShowWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.workitemCtrl, uuid.NewV4(), nil, nil)
}

func (s *WorkItem2Suite) TestWI2FailOnDelete() {
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	test.DeleteWorkitemMethodNotAllowed(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID)
}

// Temporarly disabled, See https://github.com/fabric8-services/fabric8-wit/issues/1036
func (s *WorkItem2Suite) xTestWI2SuccessDelete() {
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)

	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	test.DeleteWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID)
	test.ShowWorkitemNotFound(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
}

// TestWI2DeleteLinksOnWIDeletionOK creates two work items (WI1 and WI2) and
// creates a link between them. When one of the work items is deleted, the
// link shall be gone as well.
// Temporarly disabled, See https://github.com/fabric8-services/fabric8-wit/issues/1036
func (s *WorkItem2Suite) xTestWI2DeleteLinksOnWIDeletionOK() {
	// Create two work items (wi1 and wi2)
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "WI1"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, wi1 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), wi1)
	c.Data.Attributes[workitem.SystemTitle] = "WI2"
	_, wi2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), wi2)

	// Create link category
	linkCatPayload := newCreateWorkItemLinkCategoryPayload("test-user")
	_, linkCat := test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, linkCatPayload)
	require.NotNil(s.T(), linkCat)

	// Create link space
	spacePayload := CreateSpacePayload("test-space"+uuid.NewV4().String(), "description")
	_, space := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, spacePayload)

	// Create work item link type payload
	linkTypePayload := newCreateWorkItemLinkTypePayload("MyLinkType", *linkCat.Data.ID, *space.Data.ID)
	_, linkType := test.CreateWorkItemLinkTypeCreated(s.T(), s.svc.Context, s.svc, s.linkTypeCtrl, *space.Data.ID, linkTypePayload)
	require.NotNil(s.T(), linkType)

	// Create link between wi1 and wi2
	linkPayload := newCreateWorkItemLinkPayload(*wi1.Data.ID, *wi2.Data.ID, *linkType.Data.ID)
	_, workItemLink := test.CreateWorkItemLinkCreated(s.T(), s.svc.Context, s.svc, s.linkCtrl, linkPayload)
	require.NotNil(s.T(), workItemLink)

	// Delete work item wi1
	test.DeleteWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi1.Data.ID)

	// Check that the link was deleted by deleting wi1
	test.ShowWorkItemLinkNotFound(s.T(), s.svc.Context, s.svc, s.linkCtrl, *workItemLink.Data.ID, nil, nil)

	// Check that we can query for wi2 without problems
	test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi2.Data.ID, nil, nil)
}

func (s *WorkItem2Suite) TestWI2CreateWithArea() {
	// given
	_, areaInstance := createSpaceAndArea(s.T(), gormapplication.NewGormDB(s.DB))
	areaID := areaInstance.ID.String()
	arType := area.APIStringTypeAreas
	// when
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: newRelationBaseType(space.SystemSpace, workitem.SystemBug),
		Space:    app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		Area: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &arType,
				ID:   &areaID,
			},
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data.Relationships.Area)
	assert.Equal(s.T(), areaID, *wi.Data.Relationships.Area.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateWithArea() {
	// given
	_, areaInstance := createSpaceAndArea(s.T(), gormapplication.NewGormDB(s.DB))
	areaID := areaInstance.ID.String()
	arType := area.APIStringTypeAreas
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = &app.RelationBaseType{
		Data: &app.BaseTypeData{
			Type: "workitemtypes",
			ID:   workitem.SystemBug,
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), wi.Data.Relationships.Area)
	require.NotNil(s.T(), wi.Data.Relationships.Area.Data.ID)
	// should get root area's id for that space
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance, err := spaceRepo.Load(s.svc.Context, *c.Data.Relationships.Space.Data.ID)
	require.Nil(s.T(), err)
	areaRepo := area.NewAreaRepository(s.DB)
	rootArea, err := areaRepo.Root(context.Background(), spaceInstance.ID)
	require.Nil(s.T(), err)
	require.Equal(s.T(), rootArea.ID.String(), *wi.Data.Relationships.Area.Data.ID)
	// when
	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Area: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &arType,
				ID:   &areaID,
			},
		},
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &u)
	// then
	require.NotNil(s.T(), wiu.Data.Relationships.Area)
	require.NotNil(s.T(), wiu.Data.Relationships.Area.Data)
	assert.Equal(s.T(), areaID, *wiu.Data.Relationships.Area.Data.ID)
	assert.Equal(s.T(), arType, *wiu.Data.Relationships.Area.Data.Type)
}

func (s *WorkItem2Suite) TestWI2UpdateWithRootAreaIfMissing() {
	// given
	testSpace, rootArea := createSpaceAndArea(s.T(), gormapplication.NewGormDB(s.DB))
	log.Info(nil, nil, "creating child area...")
	childArea := area.Area{
		Name:    "Child Area of " + rootArea.Name,
		SpaceID: testSpace.ID,
		Path:    path.Path{rootArea.ID},
	}
	areaRepo := area.NewAreaRepository(s.DB)
	err := areaRepo.Create(s.ctx, &childArea)
	require.Nil(s.T(), err)
	log.Info(nil, nil, "child area created")
	childAreaID := childArea.ID.String()
	childAreaType := area.APIStringTypeAreas
	payload := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "Title",
				workitem.SystemState: workitem.SystemStateNew,
			},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(testSpace.ID, rest.AbsoluteURL(&goa.RequestData{
					Request: &http.Request{Host: "api.service.domain.org"},
				}, app.SpaceHref(testSpace.ID.String()))),
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   workitem.SystemBug,
					},
				},
				Area: &app.RelationGeneric{
					Data: &app.GenericData{
						Type: &childAreaType,
						ID:   &childAreaID,
					},
				},
			},
		},
	}
	s.T().Log("Space ID:", testSpace.ID.String())
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, testSpace.ID, &payload)
	require.NotNil(s.T(), wi.Data.Relationships.Space)
	require.NotNil(s.T(), wi.Data.Relationships.Space.Data.ID)
	require.NotNil(s.T(), wi.Data.Relationships.Area)
	require.NotNil(s.T(), wi.Data.Relationships.Area.Data.ID)
	// when
	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Area: &app.RelationGeneric{},
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &u)
	// then
	require.NotNil(s.T(), wiu.Data.Relationships.Space)
	require.NotNil(s.T(), wiu.Data.Relationships.Space.Data)
	require.NotNil(s.T(), wiu.Data.Relationships.Area)
	require.NotNil(s.T(), wiu.Data.Relationships.Area.Data)
	// should be in the same space
	assert.Equal(s.T(), testSpace.ID, *wiu.Data.Relationships.Space.Data.ID)
	// should have been set to root area
	assert.Equal(s.T(), rootArea.ID.String(), *wiu.Data.Relationships.Area.Data.ID)
}

func (s *WorkItem2Suite) TestWI2CreateUnknownArea() {
	// given
	arType := area.APIStringTypeAreas
	areaID := uuid.NewV4().String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Area = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &arType,
			ID:   &areaID,
		},
	}
	// when/then
	test.CreateWorkitemsNotFound(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2CreateWithIteration() {
	// given
	_, _, _, _, iterationInstance := createSpaceAndRootAreaAndIterations(s.T(), gormapplication.NewGormDB(s.DB))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration
	// when
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: newRelationBaseType(space.SystemSpace, workitem.SystemBug),
		Space:    app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		Iteration: &app.RelationGeneric{
			Data: &app.GenericData{
				Type: &itType,
				ID:   &iterationID,
			},
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	assert.Equal(s.T(), iterationID, *wi.Data.Relationships.Iteration.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateWithIteration() {
	// given
	_, _, _, _, iterationInstance := createSpaceAndRootAreaAndIterations(s.T(), gormapplication.NewGormDB(s.DB))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	assert.NotNil(s.T(), wi.Data.Relationships.Iteration)
	// should get root iteration's id for that space
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance, err := spaceRepo.Load(s.svc.Context, *c.Data.Relationships.Space.Data.ID)
	iterationRepo := iteration.NewIterationRepository(s.DB)
	rootIteration, err := iterationRepo.Root(context.Background(), spaceInstance.ID)
	require.Nil(s.T(), err)
	require.Equal(s.T(), rootIteration.ID.String(), *wi.Data.Relationships.Iteration.Data.ID)
	// when
	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
		},
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &u)
	// then
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration)
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration.Data)
	assert.Equal(s.T(), iterationID, *wiu.Data.Relationships.Iteration.Data.ID)
	assert.Equal(s.T(), itType, *wiu.Data.Relationships.Iteration.Data.Type)
}

func (s *WorkItem2Suite) TestWI2UpdateWithRootIterationIfMissing() {
	// given
	testSpace, _, rootIteration, _, otherIteration := createSpaceAndRootAreaAndIterations(s.T(), gormapplication.NewGormDB(s.DB))
	iterationID := otherIteration.ID.String()
	iterationType := iteration.APIStringTypeIteration
	payload := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "Title",
				workitem.SystemState: workitem.SystemStateNew,
			},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(testSpace.ID, rest.AbsoluteURL(&goa.RequestData{
					Request: &http.Request{Host: "api.service.domain.org"},
				}, app.SpaceHref(testSpace.ID.String()))),
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   workitem.SystemBug,
					},
				},
				Iteration: &app.RelationGeneric{
					Data: &app.GenericData{
						Type: &iterationType,
						ID:   &iterationID,
					},
				},
			},
		},
	}
	s.T().Log("Space ID:", testSpace.ID.String())
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, testSpace.ID, &payload)
	require.NotNil(s.T(), wi.Data.Relationships.Space)
	require.NotNil(s.T(), wi.Data.Relationships.Space.Data.ID)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration.Data.ID)
	// when
	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships = &app.WorkItemRelationships{
		Iteration: &app.RelationGeneric{},
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &u)
	// then
	require.NotNil(s.T(), wiu.Data.Relationships.Space)
	require.NotNil(s.T(), wiu.Data.Relationships.Space.Data)
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration)
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration.Data)
	// should be in the same space
	assert.Equal(s.T(), testSpace.ID, *wiu.Data.Relationships.Space.Data.ID)
	// should have been set to root iteration
	assert.Equal(s.T(), rootIteration.ID.String(), *wiu.Data.Relationships.Iteration.Data.ID)
}

func (s *WorkItem2Suite) TestWI2UpdateRemoveIteration() {
	s.T().Skip("iteration.data can't be sent as nil from client libs since it's optionall and is removed during json encoding")
	// given
	_, _, _, _, iterationInstance := createSpaceAndRootAreaAndIterations(s.T(), gormapplication.NewGormDB(s.DB))
	iterationID := iterationInstance.ID.String()
	itType := iteration.APIStringTypeIteration
	// when
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
		},
	}
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	assert.NotNil(s.T(), wi.Data.Relationships.Iteration)
	assert.NotNil(s.T(), wi.Data.Relationships.Iteration.Data)
	u := minimumRequiredUpdatePayload()
	u.Data.ID = wi.Data.ID
	u.Data.Attributes["version"] = wi.Data.Attributes["version"]
	u.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: nil,
	}
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, &u)
	// then
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration)
	assert.Nil(s.T(), wiu.Data.Relationships.Iteration.Data)
}

func (s *WorkItem2Suite) TestWI2CreateUnknownIteration() {
	// given
	itType := iteration.APIStringTypeIteration
	iterationID := uuid.NewV4().String()
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	c.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
		},
	}
	// when/then
	test.CreateWorkitemsNotFound(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithLegacyDescription() {
	// given
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := "<img src=x onerror=alert('description') />"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	_, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), fetchedWI.Data)
	require.NotNil(s.T(), fetchedWI.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWI.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), html.EscapeString(description), fetchedWI.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithPlainTextDescription() {
	// given
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := rendering.NewMarkupContent("<img src=x onerror=alert('description') />", rendering.SystemMarkupPlainText)
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	_, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), fetchedWI.Data)
	require.NotNil(s.T(), fetchedWI.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWI.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), html.EscapeString(description.Content), fetchedWI.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestWI2SuccessCreateAndPreventJavascriptInjectionWithMarkdownDescription() {
	// given
	c := minimumRequiredCreatePayload()
	title := "<img src=x onerror=alert('title') />"
	description := rendering.NewMarkupContent("<img src=x onerror=alert('description') />", rendering.SystemMarkupMarkdown)
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemDescription] = description
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// when
	_, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), fetchedWI.Data)
	require.NotNil(s.T(), fetchedWI.Data.Attributes)
	assert.Equal(s.T(), html.EscapeString(title), fetchedWI.Data.Attributes[workitem.SystemTitle])
	assert.Equal(s.T(), "<p>"+html.EscapeString(description.Content)+"</p>\n", fetchedWI.Data.Attributes[workitem.SystemDescriptionRendered])
}

func (s *WorkItem2Suite) TestCreateWIWithCodebase() {
	// given
	c := minimumRequiredCreatePayload()
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	branch := "earth-recycle-101"
	repo := "https://github.com/pranavgore09/go-tutorial.git"
	file := "main.go"
	line := 200
	cbase := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	require.NotNil(s.T(), createdWI)
	// when
	_, fetchedWI := test.ShowWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *createdWI.Data.ID, nil, nil)
	// then
	require.NotNil(s.T(), fetchedWI.Data)
	require.NotNil(s.T(), fetchedWI.Data.Attributes)
	assert.Equal(s.T(), title, fetchedWI.Data.Attributes[workitem.SystemTitle])
	cb := fetchedWI.Data.Attributes[workitem.SystemCodebase].(codebase.Content)
	assert.Equal(s.T(), repo, cb.Repository)
	assert.Equal(s.T(), branch, cb.Branch)
	assert.Equal(s.T(), file, cb.FileName)
	assert.Equal(s.T(), line, cb.LineNumber)
	assert.NotEmpty(s.T(), cb.CodebaseID)

	require.NotNil(s.T(), fetchedWI.Data.Links)
	expectedURL := fmt.Sprintf("/api/codebases/%s/edit", cb.CodebaseID)
	assert.Contains(s.T(), *fetchedWI.Data.Links.EditCodebase, expectedURL)
}

// this test aims at checking different codebaseIDs for
// two CodebaseContent with same Repository but in two different spaces
func (s *WorkItem2Suite) TestCodebaseWithSameRepoAcrossSpace() {
	// create one space
	spaceInstance, _, _ := createSpaceWithDefaults(s.svc.Context, s.DB)
	space1ID := spaceInstance.ID
	assert.NotEqual(s.T(), uuid.Nil, space1ID)

	// create a WI in above space
	// this WI should get a new CodebaseID for itself
	c := minimumRequiredCreatePayload()
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	c.Data.Relationships.Space = app.NewSpaceRelation(space1ID, "")
	branch := "earth-recycle-101"
	repo := "https://github.com/pranavgore09/go-tutorial.git"
	file := "main.go"
	line := 200
	cbase := codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	_, createdWI := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space1ID, &c)
	require.NotNil(s.T(), createdWI)
	cb := createdWI.Data.Attributes[workitem.SystemCodebase].(codebase.Content)
	codebaseID1 := cb.CodebaseID
	require.NotEmpty(s.T(), codebaseID1)

	// create another space
	spaceInstance2, _, _ := createSpaceWithDefaults(s.svc.Context, s.DB)
	space2ID := spaceInstance2.ID
	assert.NotEqual(s.T(), uuid.Nil, space2ID)

	// create a WI in new space with same Repo value
	// this WI should get a new CodebaseID for itself and not the same as before
	c = minimumRequiredCreatePayload()
	title = "Antoher solution on global warming"
	branch = "earth-recycle-102"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	c.Data.Relationships.Space = &app.RelationSpaces{Data: &app.RelationSpacesData{
		ID: &space2ID,
	}}
	cbase = codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	_, createdWI2 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space2ID, &c)
	require.NotNil(s.T(), createdWI2)
	cb2 := createdWI2.Data.Attributes[workitem.SystemCodebase].(codebase.Content)
	codebaseID2 := cb2.CodebaseID
	require.NotEmpty(s.T(), codebaseID1)
	// Repo name was same but two different Codebases are created for each
	// because items are in different sapce.
	assert.NotEqual(s.T(), codebaseID1, codebaseID2)

	// create another work item in first space
	// and check that it gets codebaseID equal to codebaseID1
	c = minimumRequiredCreatePayload()
	title = "One antoher solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	c.Data.Relationships.Space = &app.RelationSpaces{Data: &app.RelationSpacesData{
		ID: &space1ID,
	}}
	cbase = codebase.Content{
		Branch:     branch,
		Repository: repo,
		FileName:   file,
		LineNumber: line,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	_, createdWI3 := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space1ID, &c)
	require.NotNil(s.T(), createdWI3)
	cb3 := createdWI3.Data.Attributes[workitem.SystemCodebase].(codebase.Content)
	codebaseID3 := cb3.CodebaseID
	assert.Equal(s.T(), codebaseID3, codebaseID1)
}

func (s *WorkItem2Suite) TestFailToCreateWIWithCodebase() {
	// try creating WI without `Repo` : should fail
	c := minimumRequiredCreatePayload()
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	branch := "earth-recycle-101"
	cbase := codebase.Content{
		Branch: branch,
	}
	c.Data.Attributes[workitem.SystemCodebase] = cbase.ToMap()
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)

	// try creating WI with invalid GIT Repo : should fail
	c2 := minimumRequiredCreatePayload()
	title2 := "Solution on global warming"
	c2.Data.Attributes[workitem.SystemTitle] = title2
	c2.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c2.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemPlannerItem)
	branch2 := "earth-recycle-101"
	repo2 := "git://non-git.com/pranav/someproject"
	cbase2 := codebase.Content{
		Branch:     branch2,
		Repository: repo2,
	}
	c2.Data.Attributes[workitem.SystemCodebase] = cbase2.ToMap()
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c2.Data.Relationships.Space.Data.ID, &c2)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithInferredSpace() {
	// given
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// remove Space relation and see if WI gets the space out of the space URL.
	spaceID := *c.Data.Relationships.Space.Data.ID
	c.Data.Relationships.Space = nil
	// when
	_, item := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, spaceID, &c)
	// then
	require.NotNil(s.T(), item)
	assert.Equal(s.T(), title, item.Data.Attributes[workitem.SystemTitle])
	require.NotNil(s.T(), item.Data.Relationships)
	require.NotNil(s.T(), item.Data.Relationships.Space)
	assert.Equal(s.T(), space.SystemSpace, *item.Data.Relationships.Space.Data.ID)
	require.NotNil(s.T(), *item.Data.Relationships.Area)
	assert.NotNil(s.T(), *item.Data.Relationships.Area.Data.ID)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithCustomSpace() {
	// given
	spaceName := "My own Space " + uuid.NewV4().String()
	sp := &app.CreateSpacePayload{
		Data: &app.Space{
			Type: "spaces",
			Attributes: &app.SpaceAttributes{
				Name: &spaceName,
			},
		},
	}
	_, customSpace := test.CreateSpaceCreated(s.T(), s.svc.Context, s.svc, s.spaceCtrl, sp)
	require.NotNil(s.T(), customSpace)
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// set custom space and see if WI gets custom space
	c.Data.Relationships.Space.Data.ID = customSpace.Data.ID
	// when
	_, item := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), item)
	assert.Equal(s.T(), title, item.Data.Attributes[workitem.SystemTitle])
	require.NotNil(s.T(), item.Data.Relationships)
	require.NotNil(s.T(), item.Data.Relationships.Space)
	assert.Equal(s.T(), *customSpace.Data.ID, *item.Data.Relationships.Space.Data.ID)
	require.NotNil(s.T(), *item.Data.Relationships.Area)
	assert.NotNil(s.T(), *item.Data.Relationships.Area.Data.ID)
}

func (s *WorkItem2Suite) TestCreateWorkItemWithInvalidSpace() {
	// given
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// set custom space and see if WI gets custom space
	fakeSpaceID := uuid.NewV4()
	c.Data.Relationships.Space.Data.ID = &fakeSpaceID
	// when/then
	test.CreateWorkitemsBadRequest(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
}

func (s *WorkItem2Suite) TestDefaultSpaceAndIterationRelations() {
	// given
	c := minimumRequiredCreateWithType(workitem.SystemFeature)
	title := "Solution on global warming"
	c.Data.Attributes[workitem.SystemTitle] = title
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	// when
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	// then
	require.NotNil(s.T(), wi)
	require.NotNil(s.T(), wi.Data.Relationships)
	require.NotNil(s.T(), wi.Data.Relationships.Iteration)
	spaceRepo := space.NewRepository(s.DB)
	spaceInstance, err := spaceRepo.Load(s.svc.Context, space.SystemSpace)
	iterationRepo := iteration.NewIterationRepository(s.DB)
	rootIteration, err := iterationRepo.Root(context.Background(), spaceInstance.ID)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), rootIteration.ID.String(), *wi.Data.Relationships.Iteration.Data.ID)
}

// Following test verifies that UPDATE on WI by setting AREA & Iteration
// works as expected and do not alter previously set values
func (s *WorkItem2Suite) TestWI2UpdateWithAreaIterationSuccessively() {
	sp, rootArea, rootIteration, areaInstance, iterationInstance := createSpaceAndRootAreaAndIterations(s.T(), gormapplication.NewGormDB(s.DB))
	iterationID := iterationInstance.ID.String()
	areaID := areaInstance.ID.String()
	itType := iteration.APIStringTypeIteration
	arType := area.APIStringTypeAreas

	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
	*c.Data.Relationships.Space.Data.ID = sp.ID
	_, wiCreated := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, sp.ID, &c)
	assert.NotNil(s.T(), wiCreated.Data.Relationships.Iteration)
	require.Equal(s.T(), rootIteration.ID.String(), *wiCreated.Data.Relationships.Iteration.Data.ID)
	assert.NotNil(s.T(), wiCreated.Data.Relationships.Area)
	require.Equal(s.T(), rootArea.ID.String(), *wiCreated.Data.Relationships.Area.Data.ID)

	workItemRepo := workitem.NewWorkItemRepository(s.DB)
	wi, err := workItemRepo.LoadByID(context.Background(), *wiCreated.Data.ID)
	require.Nil(s.T(), err)

	// update iteration of WI
	u := minimumRequiredUpdatePayload()
	u.Data.ID = &wi.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title"
	u.Data.Attributes["version"] = wi.Version
	u.Data.Relationships.Iteration = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &itType,
			ID:   &iterationID,
		},
	}
	*u.Data.Relationships.Space.Data.ID = sp.ID
	_, wiu := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, wi.ID, &u)
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration)
	require.NotNil(s.T(), wiu.Data.Relationships.Iteration.Data)
	assert.Equal(s.T(), iterationID, *wiu.Data.Relationships.Iteration.Data.ID)
	assert.Equal(s.T(), itType, *wiu.Data.Relationships.Iteration.Data.Type)

	require.NotNil(s.T(), wiu.Data.Relationships.Area)
	require.NotNil(s.T(), wiu.Data.Relationships.Area.Data)
	assert.Equal(s.T(), rootArea.ID.String(), *wiu.Data.Relationships.Area.Data.ID)
	assert.Equal(s.T(), arType, *wiu.Data.Relationships.Area.Data.Type)

	// reload the WI (version value changed)
	wi, err = workItemRepo.LoadByID(context.Background(), *wiCreated.Data.ID)
	require.Nil(s.T(), err)

	// now update AREA of WI, that should not affect previously set Iteration
	u2 := minimumRequiredUpdatePayload()
	u2.Data.ID = &wi.ID
	u2.Data.Attributes[workitem.SystemTitle] = "Title"
	u2.Data.Attributes["version"] = wi.Version
	u2.Data.Relationships.Area = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &arType,
			ID:   &areaID,
		},
	}
	*u2.Data.Relationships.Space.Data.ID = sp.ID
	_, wiu2 := test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, wi.ID, &u2)
	// then
	require.NotNil(s.T(), wiu2.Data.Relationships.Area)
	require.NotNil(s.T(), wiu2.Data.Relationships.Area.Data)
	assert.Equal(s.T(), areaID, *wiu2.Data.Relationships.Area.Data.ID)
	assert.Equal(s.T(), arType, *wiu2.Data.Relationships.Area.Data.Type)

	require.NotNil(s.T(), wiu2.Data.Relationships.Iteration)
	require.NotNil(s.T(), wiu2.Data.Relationships.Iteration.Data)
	assert.Equal(s.T(), iterationID, *wiu2.Data.Relationships.Iteration.Data.ID)
	assert.Equal(s.T(), itType, *wiu2.Data.Relationships.Iteration.Data.Type)
}

//Ignore, middlewares not respected by the generated test framework. No way to modify Request?
// Require full HTTP request access.
func (s *WorkItem2Suite) xTestWI2IfModifiedSince() {
	// given
	c := minimumRequiredCreatePayload()
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	c.Data.Relationships = &app.WorkItemRelationships{
		BaseType: newRelationBaseType(space.SystemSpace, workitem.SystemBug),
	}
	resp, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
	lastMod := resp.Header().Get("Last-Modified")
	s.svc.Use(func(handler goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) (err error) {
			req.Header.Set("If-Modified-Since", lastMod)
			return nil
		}
	})
	// when/then
	test.ShowWorkitemNotModified(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *wi.Data.ID, nil, nil)
}

func (s *WorkItem2Suite) TestWI2ListForChildIteration() {
	grandParentIteration := createOneRandomIteration(s.svc.Context, s.DB)
	require.NotNil(s.T(), grandParentIteration)

	parentIteration := newChildIteration(s.svc.Context, s.DB, grandParentIteration)
	require.NotNil(s.T(), parentIteration)

	childIteraiton := newChildIteration(s.svc.Context, s.DB, parentIteration)
	require.NotNil(s.T(), childIteraiton)

	// create 3 work items for grandParentIteration
	grandParentIterationID := grandParentIteration.ID.String()
	for i := 0; i < 3; i++ {
		c := minimumRequiredCreatePayload()
		c.Data.Attributes[workitem.SystemTitle] = "Title"
		c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
		c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
		c.Data.Relationships.Iteration = &app.RelationGeneric{
			Data: &app.GenericData{
				ID: &grandParentIterationID,
			},
		}
		_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
		require.NotNil(s.T(), wi.Data)
		require.NotNil(s.T(), wi.Data.ID)
		require.NotNil(s.T(), wi.Data.Relationships.Iteration)
		assert.Equal(s.T(), grandParentIterationID, *wi.Data.Relationships.Iteration.Data.ID)
	}

	// create 2 work items for parentIteration
	parentIterationID := parentIteration.ID.String()
	for i := 0; i < 2; i++ {
		c := minimumRequiredCreatePayload()
		c.Data.Attributes[workitem.SystemTitle] = "Title"
		c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
		c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
		c.Data.Relationships.Iteration = &app.RelationGeneric{
			Data: &app.GenericData{
				ID: &parentIterationID,
			},
		}
		_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
		require.NotNil(s.T(), wi.Data)
		require.NotNil(s.T(), wi.Data.ID)
		require.NotNil(s.T(), wi.Data.Relationships.Iteration)
		assert.Equal(s.T(), parentIterationID, *wi.Data.Relationships.Iteration.Data.ID)
	}

	// create 2 work items for childIteraiton
	childIteraitonID := childIteraiton.ID.String()
	for i := 0; i < 2; i++ {
		c := minimumRequiredCreatePayload()
		c.Data.Attributes[workitem.SystemTitle] = "Title"
		c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
		c.Data.Relationships.BaseType = newRelationBaseType(space.SystemSpace, workitem.SystemBug)
		c.Data.Relationships.Iteration = &app.RelationGeneric{
			Data: &app.GenericData{
				ID: &childIteraitonID,
			},
		}
		_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, &c)
		require.NotNil(s.T(), wi.Data)
		require.NotNil(s.T(), wi.Data.ID)
		require.NotNil(s.T(), wi.Data.Relationships.Iteration)
		assert.Equal(s.T(), childIteraitonID, *wi.Data.Relationships.Iteration.Data.ID)
	}

	// list workitems for grandParentIteration
	_, list := test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, &grandParentIterationID, nil, nil, nil, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 7)

	// list workitems for parentIteration
	_, list = test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, &parentIterationID, nil, nil, nil, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 4)

	// list workitems for childIteraiton
	_, list = test.ListWorkitemsOK(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, space.SystemSpace, nil, nil, nil, nil, &childIteraitonID, nil, nil, nil, nil, nil, nil, nil)
	require.Len(s.T(), list.Data, 2)
}

func (s *WorkItem2Suite) TestWI2FilterExpressionRedirection() {
	c := minimumRequiredCreatePayload()
	queryExpression := fmt.Sprintf(`{"iteration" : "%s"}`, uuid.NewV4().String())
	expectedLocation := fmt.Sprintf(`/api/search?filter[expression]={"%s":[{"space": "%s" }, %s]}`, search.Q_AND, *c.Data.Relationships.Space.Data.ID, queryExpression)
	respWriter := test.ListWorkitemsTemporaryRedirect(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *c.Data.Relationships.Space.Data.ID, nil, nil, nil, &queryExpression, nil, nil, nil, nil, nil, nil, nil, nil)
	location := respWriter.Header().Get("location")
	assert.Contains(s.T(), location, expectedLocation)
}

func (s *WorkItem2Suite) TestNotificationSendOnCreate() {
	// given
	// Default created WI in setupTest
	// when

	// then
	require.Equal(s.T(), 1, len(s.notification.Messages))
	assert.Equal(s.T(), "workitem.create", s.notification.Messages[0].MessageType)
	assert.Equal(s.T(), s.wi.ID.String(), s.notification.Messages[0].TargetID)
}

func (s *WorkItem2Suite) TestNotificationSendOnUpdate() {
	// given
	// Default created WI in setupTest

	// when
	u := minimumRequiredUpdatePayload()
	u.Data.ID = s.wi.ID
	u.Data.Attributes[workitem.SystemTitle] = "Title 2"
	u.Data.Attributes[workitem.SystemVersion] = s.wi.Attributes[workitem.SystemVersion]

	test.UpdateWorkitemOK(s.T(), s.svc.Context, s.svc, s.workitemCtrl, *u.Data.ID, &u)

	// then
	require.Equal(s.T(), 2, len(s.notification.Messages))
	// index 0 is workitem.create, index 1 should be workitem.update
	assert.Equal(s.T(), "workitem.update", s.notification.Messages[1].MessageType)
	assert.Equal(s.T(), s.wi.ID.String(), s.notification.Messages[1].TargetID)
}

func minimumRequiredCreatePayloadWithSpace(spaceID uuid.UUID) app.CreateWorkitemsPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(spaceID.String()))

	return app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(spaceID, spaceSelfURL),
			},
		},
	}
}

func minimumRequiredUpdatePayloadWithSpace(spaceID uuid.UUID) app.UpdateWorkitemPayload {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(spaceID.String()))
	return app.UpdateWorkitemPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				Space: app.NewSpaceRelation(spaceID, spaceSelfURL),
			},
		},
	}
}

func (s *WorkItemSuite) TestUpdateWorkitemForSpaceCollaborator() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "TestUpdateWorkitemForSpaceCollaborator-"+uuid.NewV4().String(), "TestWI")
	require.Nil(s.T(), err)
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *testIdentity)
	// Create new workitem
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemBug, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity, &TestSpaceAuthzService{*testIdentity})
	workitemCtrl := NewWorkitemController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	workitemsCtrl := NewWorkitemsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "TestUpdateWorkitemForSpaceCollaborator-"+uuid.NewV4().String(), "TestWI")
	svcNotAuthorized := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity2, &TestSpaceAuthzService{*testIdentity})
	workitemCtrlNotAuthorized := NewWorkitemController(svcNotAuthorized, gormapplication.NewGormDB(s.DB), s.Configuration)
	workitemsCtrlNotAuthorized := NewWorkitemsController(svcNotAuthorized, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	// Not a space owner is not authorized to create
	test.CreateWorkitemsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, workitemsCtrlNotAuthorized, *payload.Data.Relationships.Space.Data.ID, &payload)

	// Update the workitem by space collaborator
	wi.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	payload2 := minimumRequiredUpdatePayloadWithSpace(*space.ID)
	payload2.Data.ID = wi.Data.ID
	payload2.Data.Attributes = wi.Data.Attributes
	_, updated := test.UpdateWorkitemOK(s.T(), svc.Context, svc, workitemCtrl, *wi.Data.ID, &payload2)

	assert.Equal(s.T(), *wi.Data.ID, *updated.Data.ID)
	assert.Equal(s.T(), (s.wi.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(s.T(), wi.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])

	// A not-space collaborator can create a work item in a space which belongs to the openshiftio test identity
	openshiftioTestIdentityID, err := uuid.FromString("7b50ddb4-5e12-4031-bca7-3b88f92e2339")
	require.Nil(s.T(), err)
	openshiftioTestIdentity := account.Identity{
		Username:     "TestUpdateWorkitemForSpaceCollaborator-" + uuid.NewV4().String(),
		ProviderType: "TestWI",
		ID:           openshiftioTestIdentityID,
	}
	err = testsupport.CreateTestIdentityForAccountIdentity(s.DB, &openshiftioTestIdentity)
	require.Nil(s.T(), err)
	openshiftioTestIdentitySpace := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, openshiftioTestIdentity)
	payload3 := minimumRequiredCreateWithTypeAndSpace(workitem.SystemBug, *openshiftioTestIdentitySpace.ID)
	payload3.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload3.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	payload3.Data.Relationships.Space.Data.ID = openshiftioTestIdentitySpace.ID
	_, wi2 := test.CreateWorkitemsCreated(s.T(), svcNotAuthorized.Context, svcNotAuthorized, workitemsCtrlNotAuthorized, *payload3.Data.Relationships.Space.Data.ID, &payload3)

	// Update the work item by the work item creator
	wi2.Data.Attributes[workitem.SystemTitle] = "Updated Test WI"
	payload4 := minimumRequiredUpdatePayloadWithSpace(*openshiftioTestIdentitySpace.ID)
	payload4.Data.ID = wi2.Data.ID
	payload4.Data.Attributes = wi2.Data.Attributes
	_, updated = test.UpdateWorkitemOK(s.T(), svcNotAuthorized.Context, svcNotAuthorized, workitemCtrlNotAuthorized, *wi2.Data.ID, &payload4)

	assert.Equal(s.T(), *wi2.Data.ID, *updated.Data.ID)
	assert.Equal(s.T(), (s.wi.Attributes["version"].(int) + 1), updated.Data.Attributes["version"])
	assert.Equal(s.T(), wi2.Data.Attributes[workitem.SystemTitle], updated.Data.Attributes[workitem.SystemTitle])

	// Check the execution order
	assert.Equal(s.T(), wi2.Data.Attributes[workitem.SystemOrder], updated.Data.Attributes[workitem.SystemOrder])

	// Not a space collaborator is not authorized to update
	test.UpdateWorkitemForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, workitemCtrlNotAuthorized, *wi.Data.ID, &payload2)
	// Not a space collaborator is not authorized to delete
	// Temporarily disabled, See https://github.com/fabric8-services/fabric8-wit/issues/1036
	// test.DeleteWorkitemForbidden(s.T(), svcNotAuthrized.Context, svcNotAuthorized, workitemCtrlNotAuthorized, *wi.Data.ID)
	// Not a space collaborator is not authorized to reorder
	payload5 := minimumRequiredReorderPayload()
	var dataArray []*app.WorkItem // dataArray contains the workitem(s) that have to be reordered
	dataArray = append(dataArray, wi.Data)
	payload5.Data = dataArray
	payload5.Position.Direction = string(workitem.DirectionTop)
	test.ReorderWorkitemsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, workitemsCtrlNotAuthorized, *space.ID, &payload5)
}

func convertWorkItemToConditionalRequestEntity(appWI app.WorkItemSingle) app.ConditionalRequestEntity {
	return workitem.WorkItem{
		ID:      *appWI.Data.ID,
		Version: appWI.Data.Attributes["version"].(int),
		Fields: map[string]interface{}{
			workitem.SystemUpdatedAt: appWI.Data.Attributes[workitem.SystemUpdatedAt].(time.Time),
		},
	}
}

func createSpaceWithDefaults(ctx context.Context, db *gorm.DB) (*space.Space, *iteration.Iteration, *area.Area) {
	spaceRepo := space.NewRepository(db)
	iterationRepo := iteration.NewIterationRepository(db)
	areaRepo := area.NewAreaRepository(db)

	newSpace := space.Space{
		Name: fmt.Sprintf("The Space %v", uuid.NewV4()),
	}
	sp, err := spaceRepo.Create(ctx, &newSpace)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to create space")
		return nil, nil, nil
	}

	// root area
	ar := &area.Area{
		Name:    sp.Name,
		SpaceID: sp.ID,
	}
	err = areaRepo.Create(ctx, ar)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": sp.ID,
			"err":      err,
		}, "failed to create root area for space.")
		return nil, nil, nil
	}

	itr := &iteration.Iteration{
		Name:    sp.Name,
		SpaceID: sp.ID,
	}
	err = iterationRepo.Create(ctx, itr)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": sp.ID,
			"err":      err,
		}, "failed to create root iteration for space.")
		return nil, nil, nil
	}
	return sp, itr, ar
}
