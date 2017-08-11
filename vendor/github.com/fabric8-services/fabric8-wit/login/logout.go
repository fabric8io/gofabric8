package login

import (
	"net/url"
	"regexp"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
)

// KeycloakLogoutService represents a keycloak logout service
type KeycloakLogoutService struct {
}

// LogoutService represents logout service interface
type LogoutService interface {
	Logout(ctx *app.LogoutLogoutContext, logoutEndpoint string, validRedirectURL string) error
}

// Logout logs out user
func (s *KeycloakLogoutService) Logout(ctx *app.LogoutLogoutContext, logoutEndpoint string, validRedirectURL string) error {
	redirect := ctx.Redirect
	referrer := ctx.RequestData.Header.Get("Referer")
	if redirect == nil {
		if referrer == "" {
			log.Error(ctx, nil, "Failed to logout. Referer Header and redirect param are both empty.")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("referer Header and redirect param are both empty (at least one should be specified)"))
		}
		redirect = &referrer
	}
	log.Info(ctx, map[string]interface{}{
		"referrer": referrer,
		"redirect": redirect,
	}, "Got Request to logout!")

	redirectURL, err := url.Parse(*redirect)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"redirect_url": redirectURL,
			"err":          err,
		}, "Failed to logout. Unable to parse redirect url.")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(err.Error()))
	}

	redirectURLStr := redirectURL.String()
	matched, err := regexp.MatchString(validRedirectURL, redirectURLStr)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"redirect_url":       redirectURLStr,
			"valid_redirect_url": validRedirectURL,
			"err":                err,
		}, "Can't match redirect URL and whitelist regex")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	if !matched {
		log.Error(ctx, map[string]interface{}{
			"redirect_url":       redirectURLStr,
			"valid_redirect_url": validRedirectURL,
		}, "Redirect URL is not valid")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("not valid redirect URL"))
	}
	logoutURL, err := url.Parse(logoutEndpoint)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"logout_endpoint": logoutEndpoint,
			"err":             err,
		}, "Failed to logout. Unable to parse logout url.")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	parameters := logoutURL.Query()
	parameters.Add("redirect_uri", redirectURLStr)
	logoutURL.RawQuery = parameters.Encode()

	ctx.ResponseData.Header().Set("Location", logoutURL.String())
	return ctx.TemporaryRedirect()
}
