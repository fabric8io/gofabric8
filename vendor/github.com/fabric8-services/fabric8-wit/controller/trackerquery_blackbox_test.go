package controller_test

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
	witrest "github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/goadesign/goa"
)

type TestTrackerQueryREST struct {
	gormtestsupport.DBTestSuite

	RwiScheduler *remoteworkitem.Scheduler

	db    *gormapplication.GormDB
	clean func()
}

func TestRunTrackerQueryREST(t *testing.T) {
	suite.Run(t, &TestTrackerQueryREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestTrackerQueryREST) SetupTest() {
	rest.RwiScheduler = remoteworkitem.NewScheduler(rest.DB)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestTrackerQueryREST) TearDownTest() {
	rest.clean()
}

func (rest *TestTrackerQueryREST) SecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Tracker-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func (rest *TestTrackerQueryREST) UnSecuredController() (*goa.Service, *TrackerController, *TrackerqueryController) {
	svc := goa.New("Tracker-Service")
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration), NewTrackerqueryController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func getTrackerQueryTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(wittoken.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.Nil(t, err)

	createTrackerQueryPayload := bytes.NewBuffer([]byte(`{"type": "github", "url": "https://api.github.com/"}`))

	return []testSecureAPI{
		// Create tracker query API with different parameters
		{
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackerqueries",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Update tracker query API with different parameters
		{
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Delete tracker query API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackerqueries/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerQueryPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker query
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackerqueries/088481764871",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (rest *TestTrackerQueryREST) TestUnauthorizeTrackerQueryCUD() {
	UnauthorizeCreateUpdateDeleteTest(rest.T(), getTrackerQueryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTrackerQuery-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerqueryController(service, rest.db, rest.RwiScheduler, rest.Configuration)
		app.MountTrackerqueryController(service, controller)
		return nil
	})
}

func (rest *TestTrackerQueryREST) TestCreateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)
	t.Log(result.ID)

	tqpayload := newCreateTrackerQueryPayload(result.ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	t.Log(tqresult)
	if tqresult.ID == "" {
		t.Error("no id")
	}
}

func (rest *TestTrackerQueryREST) TestGetTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)

	tqpayload := newCreateTrackerQueryPayload(result.ID)

	fmt.Printf("tq payload %#v", tqpayload)
	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqresult.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqresult.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.ID)
	}
	if tqr.ID != tqresult.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.ID, tqr.ID)
	}
}

func (rest *TestTrackerQueryREST) TestUpdateTrackerQuery() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)

	tqpayload := newCreateTrackerQueryPayload(result.ID)

	_, tqresult := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqresult.ID)
	_, tqr := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqresult.ID)

	if tqr == nil {
		t.Fatalf("Tracker Query '%s' not present", tqresult.ID)
	}
	if tqr.ID != tqresult.ID {
		t.Errorf("Id should be %s, but is %s", tqresult.ID, tqr.ID)
	}

	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := witrest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	payload2 := app.UpdateTrackerQueryAlternatePayload{
		Query:     tqr.Query,
		Schedule:  tqr.Schedule,
		TrackerID: result.ID,
		Relationships: &app.TrackerQueryRelationships{
			Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		},
	}

	_, updated := test.UpdateTrackerqueryOK(t, nil, nil, trackerQueryCtrl, tqr.ID, &payload2)

	if updated.ID != tqresult.ID {
		t.Errorf("Id has changed from %s to %s", tqresult.ID, updated.ID)
	}
	if updated.Query != tqresult.Query {
		t.Errorf("Query has changed from %s to %s", tqresult.Query, updated.Query)
	}
	if updated.Schedule != tqresult.Schedule {
		t.Errorf("Type has changed has from %s to %s", tqresult.Schedule, updated.Schedule)
	}
}

// This test ensures that List does not return NIL items.
func (rest *TestTrackerQueryREST) TestTrackerQueryListItemsNotNil() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)
	t.Log(result.ID)

	tqpayload := newCreateTrackerQueryPayload(result.ID)

	test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)

	_, list := test.ListTrackerqueryOK(t, nil, nil, trackerQueryCtrl)
	for _, tq := range list {
		if tq == nil {
			t.Error("Returned Tracker Query found nil")
		}
	}
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (rest *TestTrackerQueryREST) TestCreateTrackerQueryValidId() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, trackerCtrl, trackerQueryCtrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://api.github.com",
		Type: "github",
	}
	_, result := test.CreateTrackerCreated(t, svc.Context, svc, trackerCtrl, &payload)
	t.Log(result.ID)

	tqpayload := newCreateTrackerQueryPayload(result.ID)

	_, trackerquery := test.CreateTrackerqueryCreated(t, nil, nil, trackerQueryCtrl, &tqpayload)
	_, created := test.ShowTrackerqueryOK(t, nil, nil, trackerQueryCtrl, trackerquery.ID)
	if created != nil && created.ID != trackerquery.ID {
		t.Error("Failed because fetched Tracker query not same as requested. Found: ", trackerquery.ID, " Expected, ", created.ID)
	}
}

func newCreateTrackerQueryPayload(trackerID string) app.CreateTrackerQueryAlternatePayload {
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := witrest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	return app.CreateTrackerQueryAlternatePayload{
		Query:     "is:open is:issue user:arquillian author:aslakknutsen",
		Schedule:  "15 * * * * *",
		TrackerID: trackerID,
		Relationships: &app.TrackerQueryRelationships{
			Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
		},
	}
}
