package keycloak_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-tenant/keycloak"
	"github.com/stretchr/testify/assert"
)

// ignore for now, require vcr recording
func rTestPublicKey(t *testing.T) {

	keycloakConfig := keycloak.Config{
		BaseURL: "https://sso.prod-preview.openshift.io",
		Realm:   "fabric8",
	}
	u, err := keycloak.GetPublicKey(keycloakConfig)
	assert.NoError(t, err)
	assert.NotEqual(t, "", u)
}
