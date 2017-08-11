package controller_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"

	"context"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/goatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunSearchTests(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	db                             *gormapplication.GormDB
	svc                            *goa.Service
	clean                          func()
	testIdentity                   account.Identity
	wiRepo                         *workitem.GormWorkItemRepository
	controller                     *SearchController
	spaceBlackBoxTestConfiguration *config.ConfigurationData
	ctx                            context.Context
	testDir                        string
}

func (s *searchBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
	s.testDir = filepath.Join("test-files", "search")
}

func (s *searchBlackBoxTest) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)

	var err error
	// create a test identity
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "SearchBlackBoxTest user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity

	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	spaceBlackBoxTestConfiguration, err := config.GetConfigurationData()
	require.Nil(s.T(), err)
	s.spaceBlackBoxTestConfiguration = spaceBlackBoxTestConfiguration
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("WorkItemComment-Service", wittoken.NewManagerWithPrivateKey(priv), s.testIdentity)
	s.controller = NewSearchController(s.svc, gormapplication.NewGormDB(s.DB), spaceBlackBoxTestConfiguration)
}

func (s *searchBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *searchBlackBoxTest) TestSearchWorkItems() {
	// given
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := "specialwordforsearch"
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchPagination() {
	// given
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch2",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := "specialwordforsearch2"
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	// defaults in paging.go is 'pageSizeDefault = 20'
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.First)
	assert.Equal(s.T(), "http:///api/search?page[offset]=0&page[limit]=20&q=specialwordforsearch2", *sr.Links.Last)
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch2", r.Attributes[workitem.SystemTitle])
}

func (s *searchBlackBoxTest) TestSearchWithEmptyValue() {
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := ""
	spaceIDStr := space.SystemSpace.String()
	_, jerrs := test.ShowSearchBadRequest(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), jerrs)
	require.Len(s.T(), jerrs.Errors, 1)
	require.NotNil(s.T(), jerrs.Errors[0].ID)
}

func (s *searchBlackBoxTest) TestSearchWithDomainPortCombination() {
	description := "http://localhost:8080/detail/154687364529310 is related issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum", workitem.SystemState: workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `"http://localhost:8080/detail/154687364529310"`
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestSearchURLWithoutPort() {
	description := "This issue is related to http://localhost/detail/876394"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_without_port",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `"http://localhost/detail/876394"`
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnregisteredURLWithPort() {
	description := "Related to http://some-other-domain:8080/different-path/154687364529310/ok issue"
	expectedDescription := rendering.NewMarkupContentFromLegacy(description)
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	q := `http://some-other-domain:8080/different-path/`
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), description, r.Attributes[workitem.SystemDescription])
}

func (s *searchBlackBoxTest) TestUnwantedCharactersRelatedToSearchLogic() {
	expectedDescription := rendering.NewMarkupContentFromLegacy("Related to http://example-domain:8080/different-path/ok issue")

	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch_new",
			workitem.SystemDescription: expectedDescription,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	// add url: in the query, that is not expected by the code hence need to make sure it gives expected result.
	q := `http://url:some-random-other-domain:8080/different-path/`
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &spaceIDStr)
	// then
	require.NotNil(s.T(), sr.Data)
	assert.Empty(s.T(), sr.Data)
}

func (s *searchBlackBoxTest) getWICreatePayload() *app.CreateWorkitemsPayload {
	spaceRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	witRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(space.SystemSpace.String(), workitem.SystemTask.String()))
	c := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type:       APIStringTypeWorkItem,
			Attributes: map[string]interface{}{},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: APIStringTypeWorkItemType,
						ID:   workitem.SystemTask,
					},
					Links: &app.GenericLinks{
						Self:    &witRelatedURL,
						Related: &witRelatedURL,
					},
				},
				Space: app.NewSpaceRelation(space.SystemSpace, spaceRelatedURL),
			},
		},
	}
	c.Data.Attributes[workitem.SystemTitle] = "Title"
	c.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	return &c
}

func getServiceAsUser(testIdentity account.Identity) *goa.Service {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	service := testsupport.ServiceAsUser("TestSearch-Service", wittoken.NewManagerWithPrivateKey(priv), testIdentity)
	return service
}

// searchByURL copies much of the codebase from search_testing.go->ShowSearchOK
// and customises the values to add custom Host in the call.
func (s *searchBlackBoxTest) searchByURL(customHost, queryString string) *app.SearchWorkItemList {
	var resp interface{}
	var respSetter goatest.ResponseSetterFunc = func(r interface{}) { resp = r }
	newEncoder := func(io.Writer) goa.Encoder { return respSetter }
	s.svc.Encoder = goa.NewHTTPEncoder()
	s.svc.Encoder.Register(newEncoder, "*/*")
	rw := httptest.NewRecorder()
	query := url.Values{}
	u := &url.URL{
		Path:     fmt.Sprintf("/api/search"),
		RawQuery: query.Encode(),
		Host:     customHost,
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	require.Nil(s.T(), err)
	prms := url.Values{}
	prms["q"] = []string{queryString} // any value will do
	goaCtx := goa.NewContext(goa.WithAction(s.svc.Context, "SearchTest"), rw, req, prms)
	showCtx, err := app.NewShowSearchContext(goaCtx, req, s.svc)
	require.Nil(s.T(), err)
	// Perform action
	err = s.controller.Show(showCtx)
	// Validate response
	require.Nil(s.T(), err)
	require.Equal(s.T(), 200, rw.Code)
	mt, ok := resp.(*app.SearchWorkItemList)
	require.True(s.T(), ok)
	return mt
}

// verifySearchByKnownURLs performs actual tests on search result and knwonURL map
func (s *searchBlackBoxTest) verifySearchByKnownURLs(wi *app.WorkItemSingle, host, searchQuery string) {
	result := s.searchByURL(host, searchQuery)
	assert.NotEmpty(s.T(), result.Data)
	assert.Equal(s.T(), *wi.Data.ID, *result.Data[0].ID)

	known := search.GetAllRegisteredURLs()
	require.NotNil(s.T(), known)
	assert.NotEmpty(s.T(), known)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForListWI].URLRegex, host)
	assert.Contains(s.T(), known[search.HostRegistrationKeyForBoardWI].URLRegex, host)
}

// TestAutoRegisterHostURL checks if client's host is neatly registered as a KnwonURL or not
// Uses helper functions verifySearchByKnownURLs, searchByURL, getWICreatePayload
func (s *searchBlackBoxTest) TestAutoRegisterHostURL() {
	// service := getServiceAsUser(s.testIdentity)
	wiCtrl := NewWorkitemsController(s.svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	// create a WI, search by `list view URL` of newly created item
	newWI := s.getWICreatePayload()
	_, wi := test.CreateWorkitemsCreated(s.T(), s.svc.Context, s.svc, wiCtrl, *newWI.Data.Relationships.Space.Data.ID, newWI)
	require.NotNil(s.T(), wi)
	customHost := "own.domain.one"
	queryString := fmt.Sprintf("http://%s/work-item/list/detail/%d", customHost, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost, queryString)

	// Search by `board view URL` of newly created item
	customHost2 := "own.domain.two"
	queryString2 := fmt.Sprintf("http://%s/work-item/board/detail/%d", customHost2, wi.Data.Attributes[workitem.SystemNumber])
	s.verifySearchByKnownURLs(wi, customHost2, queryString2)
}

func (s *searchBlackBoxTest) TestSearchWorkItemsSpaceContext() {
	name1 := "Ultimate Space 1" + uuid.NewV4().String()
	var space1 *space.Space
	application.Transactional(s.db, func(app application.Application) error {
		sp := space.Space{
			Name: name1,
		}
		var err error
		space1, err = app.Spaces().Create(context.Background(), &sp)
		require.Nil(s.T(), err)
		return nil
	})

	name2 := "Ultimate Space 2" + uuid.NewV4().String()
	var space2 *space.Space
	application.Transactional(s.db, func(app application.Application) error {
		sp := space.Space{
			Name: name2,
		}
		var err error
		space2, err = app.Spaces().Create(context.Background(), &sp)
		require.Nil(s.T(), err)
		return nil
	})

	// WI for space 1
	for i := 0; i < 3; i++ {
		wi, err := s.wiRepo.Create(
			s.ctx,
			space1.ID,
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:       "shutter_island common_word random - " + uuid.NewV4().String(),
				workitem.SystemDescription: nil,
				workitem.SystemCreator:     "pranav",
				workitem.SystemState:       workitem.SystemStateClosed,
			},
			s.testIdentity.ID)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wi)
	}
	// WI for space 2
	for i := 0; i < 5; i++ {
		wi, err := s.wiRepo.Create(
			s.ctx,
			space2.ID,
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:       "inception common_word random - " + uuid.NewV4().String(),
				workitem.SystemDescription: nil,
				workitem.SystemCreator:     "pranav",
				workitem.SystemState:       workitem.SystemStateClosed,
			},
			s.testIdentity.ID)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wi)
	}
	// when
	q := "common_word"
	space1IDStr := space1.ID.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &space1IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 3)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 1 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "shutter_island common_word")
	}
	space2IDStr := space2.ID.String()
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, &space2IDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 5)
	for _, item := range sr.Data {
		// make sure that retrived items are from space 2 only
		assert.Contains(s.T(), item.Attributes[workitem.SystemTitle], "inception common_word")
	}

	// when searched without spaceID then it should get all related WI
	_, sr = test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, nil)
	// then
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 8)
}

func (s *searchBlackBoxTest) TestSearchWorkItemsWithoutSpaceContext() {
	name1 := "Test Space 1.1" + uuid.NewV4().String()
	var space1 *space.Space
	application.Transactional(s.db, func(app application.Application) error {
		sp := space.Space{
			Name: name1,
		}
		var err error
		space1, err = app.Spaces().Create(context.Background(), &sp)
		require.Nil(s.T(), err)
		return nil
	})

	name2 := "Test Space 2.2" + uuid.NewV4().String()
	var space2 *space.Space
	application.Transactional(s.db, func(app application.Application) error {
		sp := space.Space{
			Name: name2,
		}
		var err error
		space2, err = app.Spaces().Create(context.Background(), &sp)
		require.Nil(s.T(), err)
		return nil
	})

	// 10 WI for space 1
	for i := 0; i < 10; i++ {
		wi, err := s.wiRepo.Create(
			s.ctx,
			space1.ID,
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:       "search_by_me random - " + uuid.NewV4().String(),
				workitem.SystemDescription: nil,
				workitem.SystemCreator:     "pranav",
				workitem.SystemState:       workitem.SystemStateClosed,
			},
			s.testIdentity.ID)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wi)
	}
	// 5 WI for space 2
	for i := 0; i < 5; i++ {
		wi, err := s.wiRepo.Create(
			s.ctx,
			space2.ID,
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:       "search_by_me random - " + uuid.NewV4().String(),
				workitem.SystemDescription: nil,
				workitem.SystemCreator:     "pranav",
				workitem.SystemState:       workitem.SystemStateClosed,
			},
			s.testIdentity.ID)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wi)
	}
	q := "search_by_me"
	// search without space context
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, nil, nil, nil, &q, nil)
	require.NotEmpty(s.T(), sr.Data)
	assert.Len(s.T(), sr.Data, 15)
}

func (s *searchBlackBoxTest) TestSearchFilter() {
	// given
	_, err := s.wiRepo.Create(
		s.ctx,
		space.SystemSpace,
		workitem.SystemBug,
		map[string]interface{}{
			workitem.SystemTitle:       "specialwordforsearch",
			workitem.SystemDescription: nil,
			workitem.SystemCreator:     "baijum",
			workitem.SystemState:       workitem.SystemStateClosed,
		},
		s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	filter := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, space.SystemSpace)
	spaceIDStr := space.SystemSpace.String()
	_, sr := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
	// then
	require.NotEmpty(s.T(), sr.Data)
	r := sr.Data[0]
	assert.Equal(s.T(), "specialwordforsearch", r.Attributes[workitem.SystemTitle])
}

// It creates 1 space
// creates and adds 2 collaborators in the space
// creates 2 iterations within it
// 8 work items with different states & iterations & assignees & types
// and tests multiple combinations of space, state, iteration, assignee, type
func (s *searchBlackBoxTest) TestSearchQueryScenarioDriven() {
	spaceOwner, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestSearchQueryScenarioDriven-"), "TestWISearch")
	require.Nil(s.T(), err)

	// create 2 space collaborators' identity
	alice, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestSearchQueryScenarioDriven-"), "TestWISearch")
	require.Nil(s.T(), err)

	bob, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestSearchQueryScenarioDriven-"), "TestWISearch")
	require.Nil(s.T(), err)

	spaceInstance := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *spaceOwner)
	spaceIDStr := spaceInstance.ID.String()

	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svcWithSpaceOwner := testsupport.ServiceAsSpaceUser("Search-Service", wittoken.NewManagerWithPrivateKey(priv), *spaceOwner, &TestSpaceAuthzService{*spaceOwner})
	collaboratorRESTInstance := &TestCollaboratorsREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")}
	collaboratorRESTInstance.policy = &auth.KeycloakPolicy{
		Name:             "TestCollaborators-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	collaboratorCtrl := NewCollaboratorsController(svcWithSpaceOwner, s.db, s.Configuration, &DummyPolicyManager{rest: collaboratorRESTInstance})
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *spaceInstance.ID, alice.ID.String())
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *spaceInstance.ID, bob.ID.String())

	iterationRepo := iteration.NewIterationRepository(s.DB)
	sprint1 := iteration.Iteration{
		Name:    "Sprint 1",
		SpaceID: *spaceInstance.ID,
	}
	iterationRepo.Create(s.ctx, &sprint1)
	assert.NotEqual(s.T(), uuid.UUID{}, sprint1.ID)

	sprint2 := iteration.Iteration{
		Name:    "Sprint 2",
		SpaceID: *spaceInstance.ID,
	}
	iterationRepo.Create(s.ctx, &sprint2)
	assert.NotEqual(s.T(), uuid.UUID{}, sprint2.ID)

	wirepo := workitem.NewWorkItemRepository(s.DB)

	// create 3 WI with state "resolved" and iteration 1
	for i := 0; i < 3; i++ {
		_, err := wirepo.Create(
			s.ctx, sprint1.SpaceID, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("New issue #%d", i),
				workitem.SystemState:     workitem.SystemStateResolved,
				workitem.SystemIteration: sprint1.ID.String(),
				workitem.SystemAssignees: []string{alice.ID.String()},
			}, s.testIdentity.ID)
		require.Nil(s.T(), err)
	}

	// create 5 WI with state "closed" and iteration 2
	for i := 0; i < 5; i++ {
		_, err := wirepo.Create(
			s.ctx, sprint2.SpaceID, workitem.SystemFeature,
			map[string]interface{}{
				workitem.SystemTitle:     fmt.Sprintf("Closed issue #%d", i),
				workitem.SystemState:     workitem.SystemStateClosed,
				workitem.SystemIteration: sprint2.ID.String(),
				workitem.SystemAssignees: []string{bob.ID.String()},
			}, s.testIdentity.ID)
		require.Nil(s.T(), err)
	}

	s.T().Run("state=resolved AND iteration=sprint1", func(t *testing.T) {
		filter := fmt.Sprintf(`
			{"$AND": [
				{"state": "%s"},
				{"iteration": "%s"}
			]}`,
			workitem.SystemStateResolved, sprint1.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3) // resolved items having sprint1 are 3
	})

	s.T().Run("state=resolved AND iteration=sprint2", func(t *testing.T) {
		filter := fmt.Sprintf(`
			{"$AND": [
				{"state": "%s"},
				{"iteration": "%s"}
			]}`,
			workitem.SystemStateResolved, sprint2.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.Len(s.T(), result.Data, 0) // No items having state=resolved && sprint2
	})

	s.T().Run("state=resolved OR iteration=sprint2", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
			{"$OR": [
				{"state": "%s"},
				{"iteration": "%s"}
			]}`,
			workitem.SystemStateResolved, sprint2.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3+5) // resolved items + items in iteraion2
	})

	s.T().Run("state IN resolved, closed", func(t *testing.T) {
		// following test does not include any "space" deliberately, hence if there
		// is any work item in the test-DB having state=resolved following count
		// will fail
		filter := fmt.Sprintf(`
			{"state": {"$IN": ["%s", "%s"]}}`,
			workitem.SystemStateResolved, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3+5) // state = resolved or state = closed
	})

	s.T().Run("space=ID AND (state=resolved OR iteration=sprint2)", func(t *testing.T) {
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$OR": [
					{"state": "%s"},
					{"iteration": "%s"}
				]}
			]}`,
			spaceIDStr, workitem.SystemStateResolved, sprint2.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3+5)
	})

	s.T().Run("space=ID AND (state!=resolved AND iteration=sprint1)", func(t *testing.T) {
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$AND": [
					{"state": "%s", "negate": true},
					{"iteration": "%s"}
				]}
			]}`,
			spaceIDStr, workitem.SystemStateResolved, sprint1.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.Len(s.T(), result.Data, 0)
	})

	s.T().Run("space=ID AND (state!=open AND iteration!=fake-iterationID)", func(t *testing.T) {
		fakeIterationID1 := uuid.NewV4()
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$AND": [
					{"state": "%s", "negate": true},
					{"iteration": "%s", "negate": true}
				]}
			]}`,
			spaceIDStr, workitem.SystemStateOpen, fakeIterationID1)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 8) // all items are other than open state & in other thatn fake itr
	})

	s.T().Run("space=FakeID AND state=closed", func(t *testing.T) {
		fakeSpaceID1 := uuid.NewV4().String()
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"state": "%s"}
			]}`,
			fakeSpaceID1, workitem.SystemStateOpen)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &fakeSpaceID1)
		require.Len(s.T(), result.Data, 0) // we have 5 closed items but they are in different space
	})

	s.T().Run("space=spaceID AND state=closed AND assignee=bob", func(t *testing.T) {
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"assignee":"%s"},
				{"state": "%s"}
			]}`,
			spaceIDStr, bob.ID, workitem.SystemStateClosed)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 5) // we have 5 closed items assigned to bob
	})

	s.T().Run("space=spaceID AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see what alice did in sprint1
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"assignee":"%s"},
				{"iteration": "%s"}
			]}`,
			spaceIDStr, alice.ID, sprint1.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3) // alice worked on 3 issues in sprint1
	})

	s.T().Run("space=spaceID AND state!=closed AND iteration=sprint1 AND assignee=alice", func(t *testing.T) {
		// Let's see non-closed issues alice working on from sprint1
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"assignee":"%s"},
				{"state":"%s", "negate": true},
				{"iteration": "%s"}
			]}`,
			spaceIDStr, alice.ID, workitem.SystemStateClosed, sprint1.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3)
	})

	s.T().Run("space=spaceID AND (state=closed or state=resolved)", func(t *testing.T) {
		// get me all closed and resolved work items from my space
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$OR": [
					{"state":"%s"},
					{"state":"%s"}
				]}
			]}`,
			spaceIDStr, workitem.SystemStateClosed, workitem.SystemStateResolved)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3+5) //resolved + closed
	})

	s.T().Run("space=spaceID AND (type=bug OR type=feature)", func(t *testing.T) {
		// get me all bugs or features in myspace
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$OR": [
					{"type":"%s"},
					{"type":"%s"}
				]}
			]}`,
			spaceIDStr, workitem.SystemBug, workitem.SystemFeature)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3+5) //bugs + features
	})

	s.T().Run("space=spaceID AND (type=bug AND state=resolved AND (assignee=bob OR assignee=alice))", func(t *testing.T) {
		// get me all Resolved bugs assigned to bob or alice
		filter := fmt.Sprintf(`
			{"$AND": [
				{"space":"%s"},
				{"$AND": [
					{"$AND": [{"type":"%s"},{"state":"%s"}]},
					{"$OR": [{"assignee":"%s"},{"assignee":"%s"}]}
				]}
			]}`,
			spaceIDStr, workitem.SystemBug, workitem.SystemStateResolved, bob.ID, alice.ID)
		_, result := test.ShowSearchOK(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotEmpty(s.T(), result.Data)
		require.Len(s.T(), result.Data, 3) //resolved bugs
	})

	s.T().Run("bad expression missing curly brace", func(t *testing.T) {
		filter := fmt.Sprintf(`{"state": "0fe7b23e-c66e-43a9-ab1b-fbad9924fe7c"`)
		res, jerrs := test.ShowSearchBadRequest(s.T(), nil, nil, s.controller, &filter, nil, nil, nil, &spaceIDStr)
		require.NotNil(t, jerrs)
		require.Len(t, jerrs.Errors, 1)
		require.NotNil(t, jerrs.Errors[0].ID)
		ignoreString := "IGNORE_ME"
		jerrs.Errors[0].ID = &ignoreString
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.error.golden.json"), jerrs)
		compareWithGolden(t, filepath.Join(s.testDir, "show", "bad_expression_missing_curly_brace.headers.golden.json"), res.Header())
	})
}
