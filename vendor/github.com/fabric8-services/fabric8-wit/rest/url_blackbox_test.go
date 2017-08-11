package rest

import (
	"testing"

	"net/http"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsoluteURLOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	req := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	// HTTP
	urlStr := AbsoluteURL(req, "/testpath")
	assert.Equal(t, "http://api.service.domain.org/testpath", urlStr)

	// HTTPS
	r, err := http.NewRequest("", "https://api.service.domain.org", nil)
	require.Nil(t, err)
	req = &goa.RequestData{
		Request: r,
	}
	urlStr = AbsoluteURL(req, "/testpath2")
	assert.Equal(t, "https://api.service.domain.org/testpath2", urlStr)
}

func TestAbsoluteURLOKWithProxyForward(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	req := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}

	// HTTPS
	r, err := http.NewRequest("", "http://api.service.domain.org", nil)
	require.Nil(t, err)
	r.Header.Set("X-Forwarded-Proto", "https")
	req = &goa.RequestData{
		Request: r,
	}
	urlStr := AbsoluteURL(req, "/testpath2")
	assert.Equal(t, "https://api.service.domain.org/testpath2", urlStr)
}

func TestReplaceDomainPrefixOK(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	host, err := ReplaceDomainPrefix("api.service.domain.org", "sso")
	assert.Nil(t, err)
	assert.Equal(t, "sso.service.domain.org", host)
}

func TestReplaceDomainPrefixInTooShortHostFails(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	_, err := ReplaceDomainPrefix("org", "sso")
	assert.NotNil(t, err)
}
