package openshift_test

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-tenant/openshift"
	"github.com/stretchr/testify/assert"
)

// Ignore for now. Require VCR setup
func xTestWhoAmI(t *testing.T) {

	config := openshift.Config{
		MasterURL: "https://tsrv.devshift.net:8443",
		Token:     "rvoojTBiIOQJwATgTAIgydB7puKaHdI-RfqTmfv59nY",
	}

	u, err := openshift.WhoAmI(config)
	fmt.Println("Error: ", err)
	assert.NoError(t, err)
	assert.Equal(t, "aslak@4fs.no", u)
}
