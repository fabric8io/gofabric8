package openshift_test

import (
	"fmt"
	"testing"

	"regexp"

	"github.com/fabric8io/fabric8-init-tenant/openshift"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var processTemplate = `
- apiVersion: v1
  kind: Project
  metadata:
    annotations:
      openshift.io/description: ${PROJECT_DESCRIPTION}
      openshift.io/display-name: ${PROJECT_DISPLAYNAME}
      openshift.io/requester: ${PROJECT_REQUESTING_USER}
      serviceaccounts.openshift.io/oauth-redirectreference.jenkins: '{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"jenkins"}}'
    labels:
      provider: fabric8
      project: fabric8-online-team-environments
      version: 1.0.58
      group: io.fabric8.online.packages
    name: ${PROJECT_NAME}
`

func TestProcess(t *testing.T) {
	vars := map[string]string{
		"PROJECT_DESCRIPTION":     "Test-Project-Description",
		"PROJECT_DISPLAYNAME":     "Test-Project-Name",
		"PROJECT_REQUESTING_USER": "Aslak-User",
		"PROJECT_NAME":            "Aslak-Test",
	}

	proccsed, err := openshift.Process(processTemplate, vars)
	require.Nil(t, err, "error processing template")

	fmt.Println(proccsed)

	t.Run("verify no template markers in output", func(t *testing.T) {
		assert.False(t, regexp.MustCompile(`\${([A-Z_]+)}`).MatchString(proccsed))
		assert.False(t, regexp.MustCompile(`{{([A-Z_]+)}}`).MatchString(proccsed))
	})
	t.Run("verify markers were replaced", func(t *testing.T) {
		assert.Contains(t, proccsed, vars["PROJECT_DESCRIPTION"], "missing")
		assert.Contains(t, proccsed, vars["PROJECT_DISPLAYNAME"], "missing")
		assert.Contains(t, proccsed, vars["PROJECT_REQUESTING_USER"], "missing")
		assert.Contains(t, proccsed, vars["PROJECT_NAME"], "missing")
	})
	t.Run("Verify not fiddling with values", func(t *testing.T) {
		assert.Contains(t, proccsed, `'{"kind":"OAuthRedirectReference","apiVersion":"v1","reference":{"kind":"Route","name":"jenkins"}}'`)
	})

}
