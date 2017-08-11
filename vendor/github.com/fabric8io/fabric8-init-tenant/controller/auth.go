package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-tenant/app"
	"github.com/fabric8-services/fabric8-tenant/jsonapi"
	"github.com/fabric8-services/fabric8-tenant/keycloak"
	"github.com/fabric8-services/fabric8-tenant/openshift"
	"github.com/fabric8-services/fabric8-tenant/tenant"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// AuthController implements the auth resource.
type AuthController struct {
	*goa.Controller
	tenantService   tenant.Service
	keycloakConfig  keycloak.Config
	openshiftConfig openshift.Config
	templateVars    map[string]string
}

// NewAuthController creates a auth controller.
func NewAuthController(service *goa.Service, tenantService tenant.Service, keycloakConfig keycloak.Config, openshiftConfig openshift.Config, templateVars map[string]string) *AuthController {
	return &AuthController{
		Controller:      service.NewController("AuthController"),
		tenantService:   tenantService,
		keycloakConfig:  keycloakConfig,
		openshiftConfig: openshiftConfig,
		templateVars:    templateVars,
	}
}

// AuthToken runs the authToken action.
func (c *AuthController) AuthToken(ctx *app.AuthTokenAuthContext) error {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError("Missing JWT token"))
	}
	params := ctx.Params
	broker := params.Get("broker")
	realm := params.Get("realm")
	if len(realm) == 0 {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("realm", "missing!"))
	}
	if len(broker) == 0 {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("broker", "missing!"))
	}
	if openshift.KubernetesMode() && realm == "fabric8" && broker == "openshift-v3" {
		// For Kubernetes lets serve the tokens from Kubernetes
		// for the KeyCloak username's associated ServiceAccount
		openshiftUserToken, err := OpenshiftToken(c.keycloakConfig, c.openshiftConfig, token)
		if len(openshiftUserToken) == 0 {
			return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
		}
		/*
			result := []byte("access_token=" + openshiftUserToken + "&scope=full&token_type=bearer")
			contentType := "application/octet-stream"
		*/
		result := []byte(`{"access_token":"` + openshiftUserToken + `","expires_in":31536000,"scope":"user:full","token_type":"Bearer"}`)
		contentType := "application/octet-stream"

		ctx.ResponseData.Header().Set("Content-Type", contentType)
		ctx.ResponseData.WriteHeader(200)
		ctx.ResponseData.Length = len(result)
		_, err = ctx.ResponseData.Write(result)
		return err
	}

	// delegate to the underlying KeyCloak server
	var body []byte
	fullUrl := strings.TrimSuffix(c.keycloakConfig.BaseURL, "/") + ctx.Request.RequestURI
	req, err := http.NewRequest("GET", fullUrl, bytes.NewReader(body))
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, fmt.Errorf("Failed to forward request to KeyCloak: %v", err)))
	}
	copyHeaders := []string{"Authorization", "Content-Type", "Accept", "User-Agent", "Host", "Referrer"}
	for _, header := range copyHeaders {
		value := ctx.Request.Header.Get(header)
		if len(value) > 0 {
			req.Header.Set(header, value)
		}
	}
	client := CreateHttpClient(c.openshiftConfig)
	resp, err := client.Do(req)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, fmt.Errorf("Failed to invoke KeyCloak: %v", err)))
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	b := buf.Bytes()
	//result := string(b)
	status := resp.StatusCode
	ctx.ResponseData.Header().Set("Content-Type", "application/octet-stream")
	ctx.ResponseData.WriteHeader(status)
	ctx.ResponseData.Length = len(b)
	_, err = ctx.ResponseData.Write(b)
	return err
}

func CreateHttpClient(openshiftConfig openshift.Config) *http.Client {
	transport := openshiftConfig.HttpTransport
	if transport != nil {
		return &http.Client{
			Transport: transport,
		}
	}
	return http.DefaultClient
}
