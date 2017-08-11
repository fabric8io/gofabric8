package controller

import (
	"context"
	"time"

	"golang.org/x/oauth2"

	"net/http"
	"net/url"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
)

type loginConfiguration interface {
	GetKeycloakEndpointAuth(*goa.RequestData) (string, error)
	GetKeycloakEndpointToken(*goa.RequestData) (string, error)
	GetKeycloakAccountEndpoint(req *goa.RequestData) (string, error)
	GetKeycloakEndpointBroker(*goa.RequestData) (string, error)
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
	GetKeycloakClientID() string
	GetKeycloakSecret() string
	IsPostgresDeveloperModeEnabled() bool
	GetKeycloakTestUserName() string
	GetKeycloakTestUserSecret() string
	GetKeycloakTestUser2Name() string
	GetKeycloakTestUser2Secret() string
	GetValidRedirectURLs(*goa.RequestData) (string, error)
	GetHeaderMaxLength() int64
	GetAuthNotApprovedRedirect() string
}

const maxRecentSpacesForRPT = 10

// LoginController implements the login resource.
type LoginController struct {
	*goa.Controller
	auth               login.KeycloakOAuthService
	tokenManager       token.Manager
	configuration      loginConfiguration
	identityRepository account.IdentityRepository
}

// NewLoginController creates a login controller.
func NewLoginController(service *goa.Service, auth *login.KeycloakOAuthProvider, tokenManager token.Manager, configuration loginConfiguration, identityRepository account.IdentityRepository) *LoginController {
	return &LoginController{Controller: service.NewController("login"), auth: auth, tokenManager: tokenManager, configuration: configuration, identityRepository: identityRepository}
}

// Authorize runs the authorize action.
func (c *LoginController) Authorize(ctx *app.AuthorizeLoginContext) error {
	authEndpoint, err := c.configuration.GetKeycloakEndpointAuth(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak auth endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak auth endpoint URL")))
	}

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}

	entitlementEndpoint, err := c.configuration.GetKeycloakEndpointEntitlement(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak entitlement endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak entitlement endpoint URL")))
	}

	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak account endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	oauth := &oauth2.Config{
		ClientID:     c.configuration.GetKeycloakClientID(),
		ClientSecret: c.configuration.GetKeycloakSecret(),
		Scopes:       []string{"user:email"},
		Endpoint:     oauth2.Endpoint{AuthURL: authEndpoint, TokenURL: tokenEndpoint},
		RedirectURL:  rest.AbsoluteURL(ctx.RequestData, "/api/login/authorize"),
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.Perform(ctx, oauth, brokerEndpoint, entitlementEndpoint, profileEndpoint, whitelist, c.configuration.GetAuthNotApprovedRedirect())
}

// getEntitlementResourceRequestPayload creates the object which would have the information about which spaces/resources
// the entitlements' info would need to be fetched for.

func (c *LoginController) getEntitlementResourceRequestPayload(ctx context.Context, token *string) (*auth.EntitlementResource, error) {
	loggedInIdentityID, err := c.tokenManager.Extract(*token)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get ID from access token")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get ID from access token"))
	}

	// get the user object as well for this identity
	queryResult, err := c.identityRepository.Query(account.IdentityFilterByID(loggedInIdentityID.ID), account.IdentityWithUser())
	if err != nil || len(queryResult) == 0 {
		log.Error(ctx, map[string]interface{}{
			"err":         err,
			"identity_id": *loggedInIdentityID,
		}, "unable to query Identity")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to query Identity"))
	}
	loggedInIdentity := queryResult[0]
	contextInfoLoggedInIdentity := loggedInIdentity.User.ContextInformation
	_, recentSpacesPresent := contextInfoLoggedInIdentity["recentSpaces"]
	if contextInfoLoggedInIdentity == nil || !recentSpacesPresent {
		log.Warn(ctx, map[string]interface{}{
			"identity_id": *loggedInIdentityID,
		}, "unable to find recentSpaces in ContextInformation")
		return nil, nil
	}

	var spacesToGetEntitlementsFor []auth.ResourceSet
	recentSpaces := contextInfoLoggedInIdentity["recentSpaces"].([]interface{})
	for i, v := range recentSpaces {
		if i == maxRecentSpacesForRPT {
			log.Info(ctx, map[string]interface{}{
				"identity_id":                   *loggedInIdentityID,
				"max_recent_spaces_for_rpt":     maxRecentSpacesForRPT,
				"total_number_of_recent_spaces": len(recentSpaces),
			}, "more than the allowed maximum number of recent spaces found")
			break
		}
		recentSpaceID, ok := v.(string)
		if !ok {
			log.Warn(ctx, map[string]interface{}{
				"identity_id": *loggedInIdentityID,
			}, "unable to find a string uuid in recentSpaces in contextInformation")
			return nil, nil
		}
		spacesToGetEntitlementsFor = append(spacesToGetEntitlementsFor, auth.ResourceSet{Name: recentSpaceID}) // pass by reference?
	}
	if len(spacesToGetEntitlementsFor) == 0 {
		log.Info(ctx, map[string]interface{}{
			"identity_id": *loggedInIdentityID,
		}, "no recent spaces found for optimizing fetching of rpt")
		return nil, nil
	}
	resource := &auth.EntitlementResource{
		Permissions: spacesToGetEntitlementsFor,
	}
	log.Info(ctx, map[string]interface{}{
		"identity_id": *loggedInIdentityID,
	}, "recent spaces will be used for fetching rpt")
	return resource, nil
}

// Refresh obtain a new access token using the refresh token.
func (c *LoginController) Refresh(ctx *app.RefreshLoginContext) error {
	refreshToken := ctx.Payload.RefreshToken
	if refreshToken == nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("refresh_token", nil).Expected("not nil"))
	}

	client := &http.Client{Timeout: 10 * time.Second}
	endpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}
	res, err := client.PostForm(endpoint, url.Values{
		"client_id":     {c.configuration.GetKeycloakClientID()},
		"client_secret": {c.configuration.GetKeycloakSecret()},
		"refresh_token": {*refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "error when obtaining token")))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case 200:
		// OK
	case 401:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	case 400:
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(res.Status+" "+rest.ReadBody(res.Body)))
	default:
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.New(res.Status+" "+rest.ReadBody(res.Body))))
	}

	token, err := auth.ReadToken(ctx, res)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, err)
	}

	entitlementEndpoint, err := c.configuration.GetKeycloakEndpointEntitlement(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}

	resources, err := c.getEntitlementResourceRequestPayload(ctx, token.AccessToken)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "failed to obtain create entitlement resource request ")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}

	// Disallow fetching of all entitlements if no resources are specified
	if resources != nil {
		rpt, err := auth.GetEntitlement(ctx, entitlementEndpoint, resources, *token.AccessToken)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to obtain entitlement during login")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		if rpt != nil && int64(len(*rpt)) <= c.configuration.GetHeaderMaxLength() {
			// If the rpt token is not too long for using it as a Bearer in http requests because of header size limit
			// the swap access token for the rpt token which contains all resources available to the user
			token.AccessToken = rpt
		}
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return ctx.OK(convertToken(*token))
}

func convertToken(token auth.Token) *app.AuthToken {
	return &app.AuthToken{Token: &app.TokenData{
		AccessToken:      token.AccessToken,
		ExpiresIn:        token.ExpiresIn,
		NotBeforePolicy:  token.NotBeforePolicy,
		RefreshExpiresIn: token.RefreshExpiresIn,
		RefreshToken:     token.RefreshToken,
		TokenType:        token.TokenType,
	}}
}

// Link links identity provider(s) to the user's account
func (c *LoginController) Link(ctx *app.LinkLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.Link(ctx, brokerEndpoint, clientID, whitelist)
}

// Linksession links identity provider(s) to the user's account
func (c *LoginController) Linksession(ctx *app.LinksessionLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL")))
	}
	clientID := c.configuration.GetKeycloakClientID()
	whitelist, err := c.configuration.GetValidRedirectURLs(ctx.RequestData)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.LinkSession(ctx, brokerEndpoint, clientID, whitelist)
}

// Linkcallback redirects to original referel when Identity Provider account are linked to the user account
func (c *LoginController) Linkcallback(ctx *app.LinkcallbackLoginContext) error {
	brokerEndpoint, err := c.configuration.GetKeycloakEndpointBroker(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak broker endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak broker endpoint URL ")))
	}
	clientID := c.configuration.GetKeycloakClientID()

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return c.auth.LinkCallback(ctx, brokerEndpoint, clientID)
}

// Generate obtain the access token from Keycloak for the test user
func (c *LoginController) Generate(ctx *app.GenerateLoginContext) error {
	var tokens app.AuthTokenCollection

	tokenEndpoint, err := c.configuration.GetKeycloakEndpointToken(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to get Keycloak token endpoint URL")))
	}

	testuser, err := GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUserName(), c.configuration.GetKeycloakTestUserSecret())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUserName(),
		}, "unable to get Generate User token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token ")))
	}
	// Creates the testuser user and identity if they don't yet exist
	profileEndpoint, err := c.configuration.GetKeycloakAccountEndpoint(ctx.RequestData)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak account endpoint URL")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, err))
	}
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	testuser, err = GenerateUserToken(ctx, tokenEndpoint, c.configuration, c.configuration.GetKeycloakTestUser2Name(), c.configuration.GetKeycloakTestUser2Secret())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": c.configuration.GetKeycloakTestUser2Name(),
		}, "unable to generate test token")
		return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(ctx, errs.Wrap(err, "unable to generate test token")))
	}
	// Creates the testuser2 user and identity if they don't yet exist
	c.auth.CreateOrUpdateKeycloakUser(*testuser.Token.AccessToken, ctx, profileEndpoint)
	tokens = append(tokens, testuser)

	ctx.ResponseData.Header().Set("Cache-Control", "no-cache")
	return ctx.OK(tokens)
}

// GenerateUserToken obtains the access token from Keycloak for the user
func GenerateUserToken(ctx context.Context, tokenEndpoint string, configuration loginConfiguration, username string, userSecret string) (*app.AuthToken, error) {
	if !configuration.IsPostgresDeveloperModeEnabled() {
		log.Error(ctx, map[string]interface{}{
			"method": "Generate",
		}, "Postgres developer mode not enabled")
		return nil, errors.NewInternalError(ctx, errs.New("postgres developer mode is not enabled"))
	}

	var scopes []account.Identity
	scopes = append(scopes, test.TestIdentity)
	scopes = append(scopes, test.TestObserverIdentity)

	client := &http.Client{Timeout: 10 * time.Second}

	res, err := client.PostForm(tokenEndpoint, url.Values{
		"client_id":     {configuration.GetKeycloakClientID()},
		"client_secret": {configuration.GetKeycloakSecret()},
		"username":      {username},
		"password":      {userSecret},
		"grant_type":    {"password"},
	})
	if err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "error when obtaining token"))
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Error(ctx, map[string]interface{}{
			"response_status": res.Status,
			"response_body":   rest.ReadBody(res.Body),
			"username":        username,
		}, "unable to obtain token")
		return nil, errors.NewInternalError(ctx, errs.Errorf("unable to obtain token. Response status: %s. Responce body: %s", res.Status, rest.ReadBody(res.Body)))
	}
	token, err := auth.ReadToken(ctx, res)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"token_endpoint": res,
			"err":            err,
			"username":       username,
		}, "error when unmarshal json with access token")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "error when unmarshal json with access token"))
	}

	return convertToken(*token), nil
}
