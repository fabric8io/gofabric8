package configuration

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var reqLong *goa.RequestData
var reqShort *goa.RequestData
var config *ConfigurationData

func init() {

	// ensure that the content here is executed only once.
	reqLong = &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	reqShort = &goa.RequestData{
		Request: &http.Request{Host: "api.domain.org"},
	}
	resetConfiguration()
}

func resetConfiguration() {
	var err error
	config, err = GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

func TestOpenIDConnectPathOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	path := config.openIDConnectPath("somesufix")
	assert.Equal(t, "auth/realms/"+config.GetKeycloakRealm()+"/protocol/openid-connect/somesufix", path)
}

func TestGetKeycloakURLOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	url, err := config.getKeycloakURL(reqLong, "somepath")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.service.domain.org/somepath", url)

	url, err = config.getKeycloakURL(reqShort, "somepath2")
	assert.Nil(t, err)
	assert.Equal(t, "http://sso.domain.org/somepath2", url)
}

func TestGetKeycloakHttpsURLOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r, err := http.NewRequest("", "https://sso.domain.org", nil)
	require.Nil(t, err)
	req := &goa.RequestData{
		Request: r,
	}

	url, err := config.getKeycloakURL(req, "somepath")
	assert.Nil(t, err)
	assert.Equal(t, "https://sso.domain.org/somepath", url)
}

func TestGetKeycloakURLForTooShortHostFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	r := &goa.RequestData{
		Request: &http.Request{Host: "org"},
	}
	_, err := config.getKeycloakURL(r, "somepath")
	assert.NotNil(t, err)
}

func TestKeycloakRealmInDevModeCanBeOverridden(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_KEYCLOAK_REALM"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, devModeKeycloakRealm, config.GetKeycloakRealm())

	os.Setenv(key, "somecustomrealm")
	resetConfiguration()

	assert.Equal(t, "somecustomrealm", config.GetKeycloakRealm())
}

func TestGetLogLevelOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_LOG_LEVEL"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, defaultLogLevel, config.GetLogLevel())

	os.Setenv(key, "warning")
	resetConfiguration()

	assert.Equal(t, "warning", config.GetLogLevel())
}

func TestGetTransactionTimeoutOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_POSTGRES_TRANSACTION_TIMEOUT"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	assert.Equal(t, time.Duration(5*time.Minute), config.GetPostgresTransactionTimeout())

	os.Setenv(key, "6m")
	resetConfiguration()

	assert.Equal(t, time.Duration(6*time.Minute), config.GetPostgresTransactionTimeout())
}

func TestValidRedirectURLsInDevModeCanBeOverridden(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	key := "F8_REDIRECT_VALID"
	realEnvValue := os.Getenv(key)

	os.Unsetenv(key)
	defer func() {
		os.Setenv(key, realEnvValue)
		resetConfiguration()
	}()

	whitelist, err := config.GetValidRedirectURLs(nil)
	require.Nil(t, err)
	assert.Equal(t, devModeValidRedirectURLs, whitelist)

	os.Setenv(key, "https://someDomain.org/redirect")
	resetConfiguration()
}

func TestRedirectURLsForLocalhostRequestAreExcepted(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	// Valid if requesting prod-preview to redirect to localhost or to openshift.io
	// OR if requesting openshift to redirect to openshift.io
	// Invalid otherwise
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "http://localhost:3000/home"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "https://127.0.0.1"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io:8080/api", "https://127.0.0.1"))
	assert.True(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "https://prod-preview.openshift.io/home"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io/api", "https://openshift.io/home"))
	assert.True(t, validateRedirectURL(t, "https://api.openshift.io:8080/api", "https://openshift.io/home"))
	assert.False(t, validateRedirectURL(t, "https://api.openshift.io/api", "http://localhost:3000/api"))
	assert.False(t, validateRedirectURL(t, "https://api.prod-preview.openshift.io/api", "http://domain.com"))
	assert.False(t, validateRedirectURL(t, "https://api.openshift.io/api", "http://domain.com"))
}

func validateRedirectURL(t *testing.T, request string, redirect string) bool {
	r, err := http.NewRequest("", request, nil)
	require.Nil(t, err)
	req := &goa.RequestData{
		Request: r,
	}

	whitelist, err := config.checkLocalhostRedirectException(req)
	require.Nil(t, err)

	matched, err := regexp.MatchString(whitelist, redirect)
	require.Nil(t, err)
	return matched
}
