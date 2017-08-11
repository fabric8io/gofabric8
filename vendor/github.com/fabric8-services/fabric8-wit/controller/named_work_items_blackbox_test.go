package controller_test

import (
	"strings"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestNamedWorkItemsSuite struct {
	gormtestsupport.DBTestSuite
	db                 *gormapplication.GormDB
	testIdentity       account.Identity
	testSpace          app.Space
	svc                *goa.Service
	workitemsCtrl      *WorkitemsController
	namedWorkItemsCtrl *NamedWorkItemsController
	wi                 *app.WorkItemSingle
	clean              func()
}

func TestNamedWorkItems(t *testing.T) {
	suite.Run(t, &TestNamedWorkItemsSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestNamedWorkItemsSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "TestUpdateWorkitemForSpaceCollaborator-"+uuid.NewV4().String(), "TestWI")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity, &TestSpaceAuthzService{s.testIdentity})
	s.workitemsCtrl = NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	s.namedWorkItemsCtrl = NewNamedWorkItemsController(s.svc, gormapplication.NewGormDB(s.DB))
	s.testSpace = CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, s.testIdentity)
}

func (s *TestNamedWorkItemsSuite) TearDownTest() {
	s.clean()
}

func (s *TestNamedWorkItemsSuite) createWorkItem() *app.WorkItemSingle {
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemBug, *s.testSpace.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, s.workitemsCtrl, *s.testSpace.ID, &payload)
	return wi
}

func (s *TestNamedWorkItemsSuite) TestLookupWorkItemByNamedSpaceAndNumberOK() {
	// given
	wi := s.createWorkItem()
	// when
	res := test.ShowNamedWorkItemsMovedPermanently(s.T(), s.svc.Context, s.svc, s.namedWorkItemsCtrl, s.testIdentity.Username, *s.testSpace.Attributes.Name, wi.Data.Attributes[workitem.SystemNumber].(int))
	// then
	require.NotNil(s.T(), res.Header().Get("Location"))
	assert.True(s.T(), strings.HasSuffix(res.Header().Get("Location"), "/workitems/"+wi.Data.ID.String()))
}

func (s *TestNamedWorkItemsSuite) TestLookupWorkItemByNamedSpaceAndNumberNotFound() {
	// when/then
	test.ShowNamedWorkItemsNotFound(s.T(), s.svc.Context, s.svc, s.namedWorkItemsCtrl, "foo", "bar", 0)
}
