package login

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	errs "github.com/pkg/errors"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	coreerrors "github.com/fabric8-services/fabric8-wit/errors"
	er "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	tokencontext "github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/oauth2"
)

// NewKeycloakOAuthProvider creates a new login.Service capable of using keycloak for authorization
func NewKeycloakOAuthProvider(identities account.IdentityRepository, users account.UserRepository, tokenManager token.Manager, db application.DB) *KeycloakOAuthProvider {
	return &KeycloakOAuthProvider{
		Identities:   identities,
		Users:        users,
		TokenManager: tokenManager,
		db:           db,
	}
}

// KeycloakOAuthProvider represents a keycloak IDP
type KeycloakOAuthProvider struct {
	Identities   account.IdentityRepository
	Users        account.UserRepository
	TokenManager token.Manager
	db           application.DB
}

// KeycloakOAuthService represents keycloak OAuth service interface
type KeycloakOAuthService interface {
	Perform(ctx *app.AuthorizeLoginContext, config *oauth2.Config, brokerEndpoint string, entitlementEndpoint string, profileEndpoint string, validRedirectURL string, userNotApprovedRedirectURL string) error
	CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context, profileEndpoint string) (*account.Identity, *account.User, error)
	Link(ctx *app.LinkLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error
	LinkSession(ctx *app.LinksessionLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error
	LinkCallback(ctx *app.LinkcallbackLoginContext, brokerEndpoint string, clientID string) error
}

type linkInterface interface {
	context.Context
	jsonapi.InternalServerError
	TemporaryRedirect() error
	BadRequest(r *app.JSONAPIErrors) error
}

// keycloakTokenClaims represents standard Keycloak token claims
type keycloakTokenClaims struct {
	Name         string `json:"name"`
	Username     string `json:"preferred_username"`
	GivenName    string `json:"given_name"`
	FamilyName   string `json:"family_name"`
	Email        string `json:"email"`
	Company      string `json:"company"`
	SessionState string `json:"session_state"`
	jwt.StandardClaims
}

var allProvidersToLink = []string{"github", "openshift-v3"}

const (
	initiateLinkingParam = "initlinking"
)

// Perform performs authentication
func (keycloak *KeycloakOAuthProvider) Perform(ctx *app.AuthorizeLoginContext, config *oauth2.Config, brokerEndpoint string, entitlementEndpoint string, profileEndpoint string, validRedirectURL string, userNotApprovedRedirectURL string) error {
	state := ctx.Params.Get("state")
	code := ctx.Params.Get("code")

	log.Debug(ctx, map[string]interface{}{
		"code":  code,
		"state": state,
	}, "login request received")

	if code != "" {
		// After redirect from oauth provider
		log.Debug(ctx, map[string]interface{}{
			"code":  code,
			"state": state,
		}, "Redireced from oauth provider")

		// validate known state
		knownReferrer, err := keycloak.getReferrer(ctx, state)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"state": state,
				"err":   err,
			}, "uknown state")
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrUnauthorized("uknown state. "+err.Error()))
			return ctx.Unauthorized(jerrors)
		}

		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
		}, "referrer found")

		keycloakToken, err := config.Exchange(ctx, code)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"code": code,
				"err":  err,
			}, "keycloak exchange operation failed")
			return redirectWithError(ctx, knownReferrer, err.Error())
		}

		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
		}, "exchanged code to access token")

		_, usr, err := keycloak.CreateOrUpdateKeycloakUser(keycloakToken.AccessToken, ctx, profileEndpoint)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to create a user and KeyCloak identity using the access token")
			switch err.(type) {
			case coreerrors.UnauthorizedError:
				if userNotApprovedRedirectURL != "" {
					log.Debug(ctx, map[string]interface{}{
						"user_not_approved_redirect_url": userNotApprovedRedirectURL,
					}, "user not approved; redirecting to registration app")
					ctx.ResponseData.Header().Set("Location", userNotApprovedRedirectURL)
					return ctx.TemporaryRedirect()
				}
				return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
			}
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}

		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
			"user_name":      usr.Email,
		}, "local user created/updated")

		// redirect back to original referrel
		referrerURL, err := url.Parse(knownReferrer)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"code":           code,
				"state":          state,
				"known_referrer": knownReferrer,
				"err":            err,
			}, "failed to parse referrer")
			return redirectWithError(ctx, knownReferrer, err.Error())
		}

		err = encodeToken(ctx, referrerURL, keycloakToken)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to encode token")
			return redirectWithError(ctx, knownReferrer, err.Error())
		}
		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
			"user_name":      usr.Email,
		}, "token encoded")

		referrerStr := referrerURL.String()

		// Check if federated identities are not likned yet
		// TODO we probably won't want to check it for the existing users.
		// But we need it for now because old users still may not be linked.
		linked, err := keycloak.checkAllFederatedIdentities(ctx, keycloakToken.AccessToken, brokerEndpoint)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"err": err,
			}, "failed to check federated indentities")
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
			"user_name":      usr.Email,
			"linked":         linked,
		}, "identities links checked")

		// Return linked=true param if account has been linked to all IdPs or linked=false if not.
		if linked {
			referrerStr = referrerStr + "&linked=true"
			ctx.ResponseData.Header().Set("Location", referrerStr)
			log.Debug(ctx, map[string]interface{}{
				"code":           code,
				"state":          state,
				"known_referrer": knownReferrer,
				"user_name":      usr.Email,
				"linked":         linked,
				"referrer_str":   referrerStr,
			}, "all good; redirecting back to referrer")
			return ctx.TemporaryRedirect()
		}

		if s, err := strconv.ParseBool(referrerURL.Query().Get(initiateLinkingParam)); err != nil || !s {
			referrerStr = referrerStr + "&linked=false"
			ctx.ResponseData.Header().Set("Location", referrerStr)
			log.Debug(ctx, map[string]interface{}{
				"code":           code,
				"state":          state,
				"known_referrer": knownReferrer,
				"user_name":      usr.Email,
				"linked":         linked,
				"referrer_str":   referrerStr,
			}, "all good; redirecting back to referrer")
			return ctx.TemporaryRedirect()
		}

		referrerStr = referrerStr + "&linked=true"
		log.Debug(ctx, map[string]interface{}{
			"code":           code,
			"state":          state,
			"known_referrer": knownReferrer,
			"user_name":      usr.Email,
			"linked":         linked,
		}, "linking identities...")
		return keycloak.autoLinkProvidersDuringLogin(ctx, keycloakToken.AccessToken, referrerStr)
	}

	// First time access, redirect to oauth provider
	redirect := ctx.Redirect
	referrer := ctx.RequestData.Header.Get("Referer")
	if redirect == nil {
		if referrer == "" {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest("Referer Header and redirect param are both empty. At least one should be specified."))
			return ctx.BadRequest(jerrors)
		}
		redirect = &referrer
	}

	// store referrer in a state reference to redirect later
	log.Debug(ctx, map[string]interface{}{
		"referrer": referrer,
		"redirect": redirect,
	}, "Got Request from!")

	stateID := uuid.NewV4()
	if ctx.Link != nil && *ctx.Link {
		// We need to save the "link" param so we don't lose it when redirect to sso for auth and back to core.
		// TODO find a better place to save this param between redirects.
		linkURL, err := url.Parse(*redirect)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(err.Error()))
		}
		parameters := linkURL.Query()
		parameters.Add(initiateLinkingParam, strconv.FormatBool(*ctx.Link))
		linkURL.RawQuery = parameters.Encode()
		s := linkURL.String()
		redirect = &s
	}
	err := keycloak.saveReferrer(ctx, stateID, *redirect, validRedirectURL)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state":    stateID,
			"referrer": referrer,
			"redirect": redirect,
			"err":      err,
		}, "unable to save the state")
		return err
	}

	redirectURL := config.AuthCodeURL(stateID.String(), oauth2.AccessTypeOnline)

	ctx.ResponseData.Header().Set("Location", redirectURL)
	return ctx.TemporaryRedirect()
}

func (keycloak *KeycloakOAuthProvider) autoLinkProvidersDuringLogin(ctx *app.AuthorizeLoginContext, token string, referrerURL string) error {
	// Link all available Identity Providers
	linkURL, err := url.Parse(rest.AbsoluteURL(ctx.RequestData, "/api/login/linksession"))
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	claims, err := parseToken(token, keycloak.TokenManager.PublicKey())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to parse token")
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}
	parameters := url.Values{}
	parameters.Add("redirect", referrerURL)
	parameters.Add("sessionState", fmt.Sprintf("%v", claims.SessionState))
	linkURL.RawQuery = parameters.Encode()
	ctx.ResponseData.Header().Set("Location", linkURL.String())
	return ctx.TemporaryRedirect()
}

// checkAllFederatedIdentities returns false if there is at least one federated identity not linked to the account
func (keycloak *KeycloakOAuthProvider) checkAllFederatedIdentities(ctx context.Context, token string, brokerEndpoint string) (bool, error) {
	for _, provider := range allProvidersToLink {
		linked, err := keycloak.checkFederatedIdentity(ctx, token, brokerEndpoint, provider)
		if err != nil {
			return false, err
		}
		if !linked {
			return false, nil
		}
	}
	return true, nil
}

// checkFederatedIdentity returns true if the account is already linked to the identity provider
func (keycloak *KeycloakOAuthProvider) checkFederatedIdentity(ctx context.Context, token string, brokerEndpoint string, provider string) (bool, error) {
	req, err := http.NewRequest("GET", brokerEndpoint+"/"+provider+"/token", nil)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err.Error(),
		}, "Unable to create http request")
		return false, er.NewInternalError(ctx, errs.Wrap(err, "unable to create http request"))
	}
	req.Header.Add("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"provider": provider,
			"err":      err.Error(),
		}, "Unable to obtain a federated identity token")
		return false, er.NewInternalError(ctx, errs.Wrap(err, "unable to obtain a federated identity token"))
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK, nil
}

// Link links identity provider(s) to the user's account using user's access token
func (keycloak *KeycloakOAuthProvider) Link(ctx *app.LinkLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error {
	token := goajwt.ContextJWT(ctx)
	claims := token.Claims.(jwt.MapClaims)
	sessionState := claims["session_state"]
	if sessionState == nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("Session state is missing in token"))
	}
	ss := sessionState.(*string)
	return keycloak.linkAccountToProviders(ctx, ctx.RequestData, ctx.ResponseData, ctx.Redirect, ctx.Provider, *ss, brokerEndpoint, clientID, validRedirectURL)
}

// LinkSession links identity provider(s) to the user's account using session state
func (keycloak *KeycloakOAuthProvider) LinkSession(ctx *app.LinksessionLoginContext, brokerEndpoint string, clientID string, validRedirectURL string) error {
	if ctx.SessionState == nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest("Authorization header or session state param is required"))
	}
	return keycloak.linkAccountToProviders(ctx, ctx.RequestData, ctx.ResponseData, ctx.Redirect, ctx.Provider, *ctx.SessionState, brokerEndpoint, clientID, validRedirectURL)
}

func (keycloak *KeycloakOAuthProvider) linkAccountToProviders(ctx linkInterface, req *goa.RequestData, res *goa.ResponseData, redirect *string, provider *string, sessionState string, brokerEndpoint string, clientID string, validRedirectURL string) error {
	referrer := req.Header.Get("Referer")

	rdr := redirect
	if rdr == nil {
		rdr = &referrer
	}

	state := uuid.NewV4()
	err := keycloak.saveReferrer(ctx, state, *rdr, validRedirectURL)
	if err != nil {
		return err
	}

	if provider != nil {
		return keycloak.linkProvider(ctx, req, res, state.String(), sessionState, *provider, nil, brokerEndpoint, clientID)
	}

	return keycloak.linkProvider(ctx, req, res, state.String(), sessionState, allProvidersToLink[0], &allProvidersToLink[1], brokerEndpoint, clientID)
}

// LinkCallback redirects to original referrer when Identity Provider account are linked to the user account
func (keycloak *KeycloakOAuthProvider) LinkCallback(ctx *app.LinkcallbackLoginContext, brokerEndpoint string, clientID string) error {
	state := ctx.State
	errorMessage := ctx.Params.Get("error")
	if state == nil {
		jsonapi.JSONErrorResponse(ctx, goa.ErrInternal("State is empty. "+errorMessage))
	}
	if errorMessage != "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(errorMessage))
	}

	next := ctx.Next
	if next != nil {
		// Link the next provider
		sessionState := ctx.SessionState
		if sessionState == nil {
			log.Error(ctx, map[string]interface{}{
				"state": state,
			}, "session state is empty")
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest("session state is empty"))
			return ctx.Unauthorized(jerrors)
		}
		providerURL, err := getProviderURL(ctx.RequestData, *state, *sessionState, *next, nextProvider(*next), brokerEndpoint, clientID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
		}
		ctx.ResponseData.Header().Set("Location", providerURL)
		return ctx.TemporaryRedirect()
	}

	// No more providers to link. Redirect back to the original referrer
	originalReferrer, err := keycloak.getReferrer(ctx, *state)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "uknown state")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrUnauthorized("uknown state. "+err.Error()))
		return ctx.Unauthorized(jerrors)
	}

	ctx.ResponseData.Header().Set("Location", originalReferrer)
	return ctx.TemporaryRedirect()
}

func nextProvider(currentProvider string) *string {
	for i, provider := range allProvidersToLink {
		if provider == currentProvider {
			if i+1 < len(allProvidersToLink) {
				return &allProvidersToLink[i+1]
			}
			return nil
		}
	}
	return nil
}

func (keycloak *KeycloakOAuthProvider) linkProvider(ctx linkInterface, req *goa.RequestData, res *goa.ResponseData, state string, sessionState string, provider string, nextProvider *string, brokerEndpoint string, clientID string) error {
	providerURL, err := getProviderURL(req, state, sessionState, provider, nextProvider, brokerEndpoint, clientID)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
	}
	res.Header().Set("Location", providerURL)
	return ctx.TemporaryRedirect()
}

func (keycloak *KeycloakOAuthProvider) saveReferrer(ctx linkInterface, state uuid.UUID, referrer string, validReferrerURL string) error {
	matched, err := regexp.MatchString(validReferrerURL, referrer)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"referrer":           referrer,
			"valid_referrer_url": validReferrerURL,
			"err":                err,
		}, "Can't match referrer and whitelist regex")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal(err.Error()))
		return ctx.InternalServerError(jerrors)
	}
	if !matched {
		log.Error(ctx, map[string]interface{}{
			"referrer":           referrer,
			"valid_referrer_url": validReferrerURL,
		}, "Referrer not valid")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrBadRequest("Not valid redirect URL"))
		return ctx.BadRequest(jerrors)
	}
	// TODO The state reference table will be collecting dead states left from some failed login attempts.
	// We need to clean up the old states from time to time.
	ref := auth.OauthStateReference{
		ID:       state,
		Referrer: referrer,
	}
	err = application.Transactional(keycloak.db, func(appl application.Application) error {
		_, err := appl.OauthStates().Create(ctx, &ref)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state":    state,
			"referrer": referrer,
			"err":      err,
		}, "unable to create oauth state reference")
		jerrors, _ := jsonapi.ErrorToJSONAPIErrors(ctx, goa.ErrInternal("Unable to create oauth state reference "+err.Error()))
		return ctx.InternalServerError(jerrors)
	}
	return nil
}

func (keycloak *KeycloakOAuthProvider) getReferrer(ctx context.Context, state string) (string, error) {
	var referrer string
	stateID, err := uuid.FromString(state)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "unable to convert oauth state to uuid")
		return "", errors.New("Unable to convert oauth state to uuid. " + err.Error())
	}
	err = application.Transactional(keycloak.db, func(appl application.Application) error {
		ref, err := appl.OauthStates().Load(ctx, stateID)
		if err != nil {
			return err
		}
		referrer = ref.Referrer
		err = appl.OauthStates().Delete(ctx, stateID)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"state": state,
			"err":   err,
		}, "unable to delete oauth state reference")
		return "", errors.New("Unable to delete oauth state reference " + err.Error())
	}
	return referrer, nil
}

func getProviderURL(req *goa.RequestData, state string, sessionState string, provider string, nextProvider *string, brokerEndpoint string, clientID string) (string, error) {
	var nextParam string
	if nextProvider != nil {
		nextParam = "&next=" + *nextProvider
	}
	callbackURL := rest.AbsoluteURL(req, "/api/login/linkcallback?provider="+provider+nextParam+"&sessionState="+sessionState+"&state="+state)

	nonce := uuid.NewV4().String()

	s := nonce + sessionState + clientID + provider
	h := sha256.New()
	h.Write([]byte(s))
	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))

	linkingURL, err := url.Parse(brokerEndpoint + "/" + provider + "/link")
	if err != nil {
		return "", err
	}

	parameters := url.Values{}
	parameters.Add("provider_id", provider)
	parameters.Add("client_id", clientID)
	parameters.Add("redirect_uri", callbackURL)
	parameters.Add("nonce", nonce)
	parameters.Add("hash", hash)
	linkingURL.RawQuery = parameters.Encode()

	return linkingURL.String(), nil
}

func numberToInt(number interface{}) (int64, error) {
	switch v := number.(type) {
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	}
	result, err := strconv.ParseInt(fmt.Sprintf("%v", number), 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func encodeToken(ctx context.Context, referrer *url.URL, outhToken *oauth2.Token) error {
	str := outhToken.Extra("expires_in")
	var expiresIn interface{}
	var refreshExpiresIn interface{}
	var err error
	expiresIn, err = numberToInt(str)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"expires_in": str,
			"err":        err,
		}, "unable to parse expires_in claim")
		return errs.WithStack(errors.New("unable to parse expires_in claim to integer: " + err.Error()))
	}
	str = outhToken.Extra("refresh_expires_in")
	refreshExpiresIn, err = numberToInt(str)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"refresh_expires_in": str,
			"err":                err,
		}, "unable to parse expires_in claim")
		return errs.WithStack(errors.New("unable to parse refresh_expires_in claim to integer: " + err.Error()))
	}
	tokenData := &app.TokenData{
		AccessToken:      &outhToken.AccessToken,
		RefreshToken:     &outhToken.RefreshToken,
		TokenType:        &outhToken.TokenType,
		ExpiresIn:        &expiresIn,
		RefreshExpiresIn: &refreshExpiresIn,
	}
	b, err := json.Marshal(tokenData)
	if err != nil {
		return errs.WithStack(errors.New("cant marshal token data struct " + err.Error()))
	}

	parameters := referrer.Query()
	parameters.Add("token_json", string(b))
	referrer.RawQuery = parameters.Encode()

	return nil
}

// CreateOrUpdateKeycloakUser creates a user and a keycloak identity. If the user and identity already exist then update them.
func (keycloak *KeycloakOAuthProvider) CreateOrUpdateKeycloakUser(accessToken string, ctx context.Context, profileEndpoint string) (*account.Identity, *account.User, error) {
	var identity *account.Identity
	var user *account.User

	claims, err := parseToken(accessToken, keycloak.TokenManager.PublicKey())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"token": accessToken,
			"err":   err,
		}, "unable to parse the token")
		return nil, nil, errors.New("unable to parse the token " + err.Error())
	}

	if err := checkClaims(claims); err != nil {
		log.Error(ctx, map[string]interface{}{
			"token": accessToken,
			"err":   err,
		}, "invalid keycloak token claims")
		return nil, nil, errors.New("invalid keycloak token claims " + err.Error())
	}

	keycloakIdentityID, _ := uuid.FromString(claims.Subject)
	identities, err := keycloak.Identities.Query(account.IdentityFilterByID(keycloakIdentityID), account.IdentityWithUser())
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"keycloak_identity_id": keycloakIdentityID,
			"err": err,
		}, "unable to  query for an identity by ID")
		return nil, nil, errors.New("Error during querying for an identity by ID " + err.Error())
	}

	if len(identities) == 0 {
		// No Identity found, create a new Identity and User
		approved, err := checkApproved(ctx, NewKeycloakUserProfileClient(), accessToken, profileEndpoint)
		if err != nil {
			return nil, nil, err
		}
		if !approved {
			return nil, nil, coreerrors.NewUnauthorizedError(fmt.Sprintf("user '%s' is not approved", claims.Username))
		}
		user = new(account.User)
		identity = &account.Identity{}
		_, err = fillUser(claims, user, identity)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"err": err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to update user/identity from claims" + err.Error())
		}
		err = application.Transactional(keycloak.db, func(appl application.Application) error {
			err := appl.Users().Create(ctx, user)
			if err != nil {
				return err
			}

			identity.ID = keycloakIdentityID
			identity.ProviderType = account.KeycloakIDP
			identity.UserID = account.NullUUID{UUID: user.ID, Valid: true}
			identity.User = *user

			err = appl.Identities().Create(ctx, identity)
			return err
		})
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"username":             claims.Username,
				"err":                  err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to create user/identity " + err.Error())
		}

	} else {
		identity = &identities[0]
		user = &identity.User
		if user.ID == uuid.Nil {
			log.Error(ctx, map[string]interface{}{
				"identity_id": keycloakIdentityID,
			}, "Found Keycloak identity is not linked to any User")
			return nil, nil, errors.New("found Keycloak identity is not linked to any User")
		}
		// let's update the existing user with the fullname, email and avatar from Keycloak,
		// in case the user changed them since the last time he/she logged in
		isChanged, err := fillUser(claims, user, identity)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"keycloak_identity_id": keycloakIdentityID,
				"err": err,
			}, "unable to create user/identity")
			return nil, nil, errors.New("failed to update user/identity from claims" + err.Error())
		} else if isChanged {
			err = application.Transactional(keycloak.db, func(appl application.Application) error {
				err = appl.Users().Save(ctx, user)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"user_id": user.ID,
						"err":     err,
					}, "unable to update user")
					return errors.New("failed to update user " + err.Error())
				}
				err = appl.Identities().Save(ctx, identity)
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"user_id": identity.ID,
						"err":     err,
					}, "unable to update identity")
					return errors.New("failed to update identity " + err.Error())
				}
				return err
			})
			if err != nil {
				log.Error(ctx, map[string]interface{}{
					"keycloak_identity_id": keycloakIdentityID,
					"username":             claims.Username,
					"err":                  err,
				}, "unable to update user/identity")
				return nil, nil, errors.New("failed to update user/identity " + err.Error())
			}
		}
	}
	return identity, user, nil
}

func checkApproved(ctx context.Context, profileService UserProfileService, accessToken string, profileEndpoint string) (bool, error) {
	profile, err := profileService.Get(ctx, accessToken, profileEndpoint)
	if err != nil {
		return false, err
	}
	if profile.Attributes == nil {
		log.Warn(ctx, map[string]interface{}{
			"username": profile.Username,
		}, "no attributes found in the user's profile")
		return false, nil
	}
	attributes := *profile.Attributes
	approved := attributes[ApprovedAttributeName]
	if len(approved) == 0 {
		log.Warn(ctx, map[string]interface{}{
			"username": profile.Username,
		}, "no approved attribute found in the user's profile or the value is empty")
		return false, nil
	}
	b, err := strconv.ParseBool(approved[0])
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"username": profile.Username,
			"approved": approved[0],
		}, "unable to parse 'approved' attribute of the user's profile")
		return false, err
	}
	if !b {
		log.Warn(ctx, map[string]interface{}{
			"username": profile.Username,
		}, "approved attribute found but set to false")
	}
	return b, nil
}

func redirectWithError(ctx *app.AuthorizeLoginContext, knownReferrer string, errorString string) error {
	ctx.ResponseData.Header().Set("Location", knownReferrer+"?error="+errorString)
	return ctx.TemporaryRedirect()
}

func parseToken(tokenString string, publicKey *rsa.PublicKey) (*keycloakTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &keycloakTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(*keycloakTokenClaims)
	if token.Valid {
		return claims, nil
	}
	return nil, errs.WithStack(errors.New("token is not valid"))
}

func generateGravatarURL(email string) (string, error) {
	if email == "" {
		return "", nil
	}
	grURL, err := url.Parse("https://www.gravatar.com/avatar/")
	if err != nil {
		return "", errs.WithStack(err)
	}
	hash := md5.New()
	hash.Write([]byte(email))
	grURL.Path += fmt.Sprintf("%v", hex.EncodeToString(hash.Sum(nil))) + ".jpg"

	// We can use our own default image if there is no gravatar available for this email
	// defaultImage := "someDefaultImageURL.jpg"
	// parameters := url.Values{}
	// parameters.Add("d", fmt.Sprintf("%v", defaultImage))
	// grURL.RawQuery = parameters.Encode()

	urlStr := grURL.String()
	return urlStr, nil
}

func checkClaims(claims *keycloakTokenClaims) error {
	if claims.Subject == "" {
		return errors.New("subject claim not found in token")
	}
	_, err := uuid.FromString(claims.Subject)
	if err != nil {
		return errors.New("subject claim from token is not UUID " + err.Error())
	}
	if claims.Username == "" {
		return errors.New("username claim not found in token")
	}
	if claims.Email == "" {
		return errors.New("email claim not found in token")
	}
	return nil
}

func fillUser(claims *keycloakTokenClaims, user *account.User, identity *account.Identity) (bool, error) {
	isChanged := false
	if user.FullName != claims.Name || user.Email != claims.Email || user.Company != claims.Company || identity.Username != claims.Username || user.ImageURL == "" {
		isChanged = true
	} else {
		return isChanged, nil
	}
	user.FullName = claims.Name
	user.Email = claims.Email
	user.Company = claims.Company
	identity.Username = claims.Username
	if user.ImageURL == "" {
		image, err := generateGravatarURL(claims.Email)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"user_full_name": user.FullName,
				"err":            err,
			}, "error when generating gravatar")
			// if there is an error, we will qualify the identity/user as unchanged.
			return false, errors.New("Error when generating gravatar " + err.Error())
		}
		user.ImageURL = image
	}
	return isChanged, nil
}

// ContextIdentity returns the identity's ID found in given context
// Uses tokenManager.Locate to fetch the identity of currently logged in user
func ContextIdentity(ctx context.Context) (*uuid.UUID, error) {
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")

		return nil, errs.New("Missing token manager")
	}
	// As mentioned in token.go, we can now safely convert tm to a token.Manager
	manager := tm.(token.Manager)
	uuid, err := manager.Locate(ctx)
	if err != nil {
		// TODO : need a way to define user as Guest
		log.Error(ctx, map[string]interface{}{
			"uuid":          uuid,
			"token_manager": manager,
			"err":           err,
		}, "identity belongs to a Guest User")

		return nil, errs.WithStack(err)
	}
	return &uuid, nil
}

// InjectTokenManager is a middleware responsible for setting up tokenManager in the context for every request.
func InjectTokenManager(tokenManager token.Manager) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctxWithTM := tokencontext.ContextWithTokenManager(ctx, tokenManager)
			return h(ctxWithTM, rw, req)
		}
	}
}
