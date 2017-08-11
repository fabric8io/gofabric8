package account

import (
	"net/http"

	"context"

	"net/url"

	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/goasupport"
	goaclient "github.com/goadesign/goa/client"
)

type tenantConfig interface {
	GetTenantServiceURL() string
}

// NewInitTenant creates a new tenant service in oso
func NewInitTenant(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return InitTenant(ctx, config)
	}
}

// NewUpdateTenant creates a new tenant service in oso
func NewUpdateTenant(config tenantConfig) func(context.Context) error {
	return func(ctx context.Context) error {
		return UpdateTenant(ctx, config)
	}
}

// InitTenant creates a new tenant service in oso
func InitTenant(ctx context.Context, config tenantConfig) error {

	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return err
	}

	c := tenant.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))

	// Ignore response for now
	_, err = c.SetupTenant(goasupport.ForwardContextRequestID(ctx), tenant.SetupTenantPath())

	return err
}

// UpdateTenant creates a new tenant service in oso
func UpdateTenant(ctx context.Context, config tenantConfig) error {

	u, err := url.Parse(config.GetTenantServiceURL())
	if err != nil {
		return err
	}

	c := tenant.New(goaclient.HTTPClientDoer(http.DefaultClient))
	c.Host = u.Host
	c.Scheme = u.Scheme
	c.SetJWTSigner(goasupport.NewForwardSigner(ctx))

	// Ignore response for now
	_, err = c.UpdateTenant(goasupport.ForwardContextRequestID(ctx), tenant.SetupTenantPath())

	return err
}
