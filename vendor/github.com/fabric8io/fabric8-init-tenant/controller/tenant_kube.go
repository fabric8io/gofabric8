package controller

import (
	"fmt"

	"github.com/fabric8-services/fabric8-tenant/app"
	"github.com/fabric8-services/fabric8-tenant/jsonapi"
	"github.com/fabric8-services/fabric8-tenant/keycloak"
	"github.com/fabric8-services/fabric8-tenant/openshift"
	"github.com/fabric8-services/fabric8-tenant/tenant"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// TenantKubeController implements the tenantKube resource.
type TenantKubeController struct {
	*goa.Controller
	tenantService   tenant.Service
	keycloakConfig  keycloak.Config
	openshiftConfig openshift.Config
	templateVars    map[string]string
}

// NewTenantKubeController creates a tenantKube controller.
func NewTenantKubeController(service *goa.Service, tenantService tenant.Service, keycloakConfig keycloak.Config, openshiftConfig openshift.Config, templateVars map[string]string) *TenantKubeController {
	return &TenantKubeController{
		Controller:      service.NewController("TenantKubeController"),
		tenantService:   tenantService,
		keycloakConfig:  keycloakConfig,
		openshiftConfig: openshiftConfig,
		templateVars:    templateVars,
	}
}

// KubeConnected checks that kubernetes tenant is connected with KeyCloak.
func (c *TenantKubeController) KubeConnected(ctx *app.KubeConnectedTenantKubeContext) error {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("Missing JWT token"))
	}

	openshiftUserToken, err := OpenshiftToken(c.keycloakConfig, c.openshiftConfig, token)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to authenticate user with keycloak")
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("Could not authorization against keycloak"))
	}

	openshiftUser, err := OpenShiftWhoAmI(token, c.openshiftConfig, openshiftUserToken)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to authenticate user with tenant target server")
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("unknown/unauthorized openshift user"))
	}

	msg := ""
	hasJenkinsNS := openshift.HasJenkinsNamespace(c.openshiftConfig, openshiftUser)
	if !hasJenkinsNS {
		fmt.Printf("\nNo Jenkins Namespace! So updating tenant\n")
		ttoken := &TenantToken{token: token}
		tenant := &tenant.Tenant{ID: ttoken.Subject(), Email: ttoken.Email()}
		exists := c.tenantService.Exists(ttoken.Subject())
		err = c.tenantService.UpdateTenant(tenant)
		if err == nil {
			tenantID := ttoken.Subject()
			tenant, err := c.tenantService.GetTenant(tenantID)
			if err == nil {
				if exists {
					err = openshift.UpdateTenant(
						ctx,
						c.keycloakConfig,
						c.openshiftConfig,
						InitTenant(ctx, c.openshiftConfig.MasterURL, c.tenantService, tenant),
						openshiftUser,
						openshiftUserToken,
						c.templateVars)
				} else {
					err = openshift.RawInitTenant(
						ctx,
						c.keycloakConfig,
						c.openshiftConfig,
						InitTenant(ctx, c.openshiftConfig.MasterURL, c.tenantService, tenant),
						openshiftUser,
						openshiftUserToken,
						c.templateVars)
				}
			}
		}
		if err != nil {
			fmt.Printf("\n failed to update tenant: %v\n", err)
			msg = fmt.Sprintf("Failed to update tenant %v", err)
		}
	} else {
		fmt.Printf("Have Jenkins namespace! So lets try check connected\n")
	}
	if err == nil {
		msg, err = openshift.KubeConnected(
			c.keycloakConfig,
			c.openshiftConfig,
			openshiftUser)
	}

	if err != nil {
		errText := fmt.Sprintf("%v", err)
		res := &app.TenantStatusSingle{
			Data: &app.TenantStatus{
				Attributes: &app.TenantStatusAttributes{
					Message: &msg,
					Error:   &errText,
				},
			},
		}
		//return ctx.Conflict(res)
		ctx.ResponseData.Header().Set("Content-Type", "application/vnd.api+json")
		return ctx.ResponseData.Service.Send(ctx.Context, 409, res)
	}
	res := &app.TenantStatusSingle{
		Data: &app.TenantStatus{
			Attributes: &app.TenantStatusAttributes{
				Message: &msg,
			},
		},
	}
	return ctx.OK(res)
}
