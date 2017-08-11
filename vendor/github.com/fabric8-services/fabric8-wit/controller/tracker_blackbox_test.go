package controller_test

import (
	"bytes"
	"net/http"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/resource"
)

type TestTrackerREST struct {
	gormtestsupport.DBTestSuite

	RwiScheduler *remoteworkitem.Scheduler

	db    *gormapplication.GormDB
	clean func()
}

func TestRunTrackerREST(t *testing.T) {
	suite.Run(t, &TestTrackerREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestTrackerREST) SetupTest() {
	rest.RwiScheduler = remoteworkitem.NewScheduler(rest.DB)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestTrackerREST) TearDownTest() {
	rest.clean()
}

func (rest *TestTrackerREST) SecuredController() (*goa.Service, *TrackerController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Tracker-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

func (rest *TestTrackerREST) UnSecuredController() (*goa.Service, *TrackerController) {
	svc := goa.New("Tracker-Service")
	return svc, NewTrackerController(svc, rest.db, rest.RwiScheduler, rest.Configuration)
}

// This test case will check authorized access to Create/Update/Delete APIs
func (rest *TestTrackerREST) TestUnauthorizeTrackerCUD() {
	UnauthorizeCreateUpdateDeleteTest(rest.T(), getTrackerTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedTracker-Service")
	}, func(service *goa.Service) error {
		controller := NewTrackerController(service, rest.db, rest.RwiScheduler, rest.Configuration)
		app.MountTrackerController(service, controller)
		return nil
	})
}

func getTrackerTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(wittoken.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	require.Nil(t, err)

	createTrackerPayload := bytes.NewBuffer([]byte(`{"type": "github", "url": "https://api.github.com/"}`))

	return []testSecureAPI{
		// Create tracker API with different parameters
		{
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                "/api/trackers",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Update tracker API with different parameters
		{
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPut,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Delete tracker API with different parameters
		{
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                "/api/trackers/12345",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createTrackerPayload,
			jwtToken:           "",
		},
		// Try fetching a random tracker
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                "/api/trackers/088481764871",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

func (rest *TestTrackerREST) TestCreateTracker() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}

	_, created := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)
	if created.ID == "" {
		t.Error("no id")
	}
}

func (rest *TestTrackerREST) TestGetTracker() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}

	_, result := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)
	test.ShowTrackerOK(t, svc.Context, svc, ctrl, result.ID)
	_, tr := test.ShowTrackerOK(t, svc.Context, svc, ctrl, result.ID)
	if tr == nil {
		t.Fatalf("Tracker '%s' not present", result.ID)
	}
	if tr.ID != result.ID {
		t.Errorf("Id should be %s, but is %s", result.ID, tr.ID)
	}

	payload2 := app.UpdateTrackerAlternatePayload{
		URL:  tr.URL,
		Type: tr.Type,
	}
	_, updated := test.UpdateTrackerOK(t, svc.Context, svc, ctrl, tr.ID, &payload2)
	if updated.ID != result.ID {
		t.Errorf("Id has changed from %s to %s", result.ID, updated.ID)
	}
	if updated.URL != result.URL {
		t.Errorf("URL has changed from %s to %s", result.URL, updated.URL)
	}
	if updated.Type != result.Type {
		t.Errorf("Type has changed has from %s to %s", result.Type, updated.Type)
	}

}

// This test ensures that List does not return NIL items.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/191
func (rest *TestTrackerREST) TestTrackerListItemsNotNil() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
	test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)

	test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)

	_, list := test.ListTrackerOK(t, svc.Context, svc, ctrl, nil, nil)

	for _, tracker := range list {
		if tracker == nil {
			t.Error("Returned Tracker found nil")
		}
	}
}

// This test ensures that ID returned by Show is valid.
// refer : https://github.com/fabric8-services/fabric8-wit/issues/189
func (rest *TestTrackerREST) TestCreateTrackerValidId() {
	t := rest.T()
	resource.Require(t, resource.Database)

	svc, ctrl := rest.SecuredController()
	payload := app.CreateTrackerAlternatePayload{
		URL:  "http://issues.jboss.com",
		Type: "jira",
	}
	_, tracker := test.CreateTrackerCreated(t, svc.Context, svc, ctrl, &payload)

	_, created := test.ShowTrackerOK(t, svc.Context, svc, ctrl, tracker.ID)
	if created != nil && created.ID != tracker.ID {
		t.Error("Failed because fetched Tracker not same as requested. Found: ", tracker.ID, " Expected, ", created.ID)
	}
}
