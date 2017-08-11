package login_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/lib/pq"
)

func TestLogout(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	configuration, err := config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	suite.Run(t, &TestLogoutSuite{configuration: configuration, logoutService: &login.KeycloakLogoutService{}})
}

type TestLogoutSuite struct {
	suite.Suite
	configuration *config.ConfigurationData
	logoutService *login.KeycloakLogoutService
}

func (s *TestLogoutSuite) SetupSuite() {
}

func (s *TestLogoutSuite) TearDownSuite() {
}

func (s *TestLogoutSuite) TestLogoutRedirectsToKeycloakWithRedirectParam() {
	s.checkRedirects("", "https://url.example.org/path", "https%3A%2F%2Furl.example.org%2Fpath")
}

func (s *TestLogoutSuite) TestLogoutRedirectsToKeycloakWithReferrer() {
	s.checkRedirects("http://openshift.io/home", "https://url.example.org/path", "http%3A%2F%2Fopenshift.io%2Fhome")
}

func (s *TestLogoutSuite) checkRedirects(redirectParam string, referrerURL string, expectedRedirectParam string) {
	rw := httptest.NewRecorder()
	u := &url.URL{
		Path: "/api/logout",
	}
	req, err := http.NewRequest("GET", u.String(), nil)
	require.Nil(s.T(), err)
	req.Header.Add("referer", referrerURL)

	prms := url.Values{}
	if redirectParam != "" {
		prms.Add("redirect", redirectParam)
	}
	ctx := context.Background()
	goaCtx := goa.NewContext(goa.WithAction(ctx, "LogoutTest"), rw, req, prms)
	logoutCtx, err := app.NewLogoutLogoutContext(goaCtx, req, goa.New("LogoutService"))
	require.Nil(s.T(), err)

	r := &goa.RequestData{
		Request: &http.Request{Host: "api.domain.io"},
	}
	logoutEndpoint, err := s.configuration.GetKeycloakEndpointLogout(r)
	require.Nil(s.T(), err)
	validURLs, err := s.configuration.GetValidRedirectURLs(r)
	require.Nil(s.T(), err)

	err = s.logoutService.Logout(logoutCtx, logoutEndpoint, validURLs)

	assert.Equal(s.T(), 307, rw.Code)
	assert.Equal(s.T(), fmt.Sprintf("%s?redirect_uri=%s", logoutEndpoint, expectedRedirectParam), rw.Header().Get("Location"))
}
