// Package authz contains the code that authorizes space operations
package authz

import (
	"context"
	"net/http"
	"time"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	tokencontext "github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/token"
	errs "github.com/pkg/errors"

	contx "context"

	"github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
)

// AuthzService represents a space authorization service
type AuthzService interface {
	Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error)
	Configuration() AuthzConfiguration
}

// AuthzConfiguration represents a Keycloak entitlement endpoint configuration
type AuthzConfiguration interface {
	GetKeycloakEndpointEntitlement(*goa.RequestData) (string, error)
}

// AuthzServiceManager represents a space autharizarion service
type AuthzServiceManager interface {
	AuthzService() AuthzService
	EntitlementEndpoint() string
}

// KeycloakAuthzServiceManager is a keyaloak implementation of a space autharizarion service
type KeycloakAuthzServiceManager struct {
	Service             AuthzService
	entitlementEndpoint string
}

// AuthzService returns a space autharizarion service
func (m *KeycloakAuthzServiceManager) AuthzService() AuthzService {
	return m.Service
}

// EntitlementEndpoint returns a keycloak entitlement endpoint URL
func (m *KeycloakAuthzServiceManager) EntitlementEndpoint() string {
	return m.entitlementEndpoint
}

// KeycloakAuthzService implements AuthzService interface
type KeycloakAuthzService struct {
	config AuthzConfiguration
	db     application.DB
}

// NewAuthzService constructs a new KeycloakAuthzService
func NewAuthzService(config AuthzConfiguration, db application.DB) *KeycloakAuthzService {
	return &KeycloakAuthzService{config: config, db: db}
}

// Configuration returns authz service configuration
func (s *KeycloakAuthzService) Configuration() AuthzConfiguration {
	return s.config
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func (s *KeycloakAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error) {
	jwttoken := goajwt.ContextJWT(ctx)
	if jwttoken == nil {
		return false, errors.NewUnauthorizedError("missing token")
	}
	tm := tokencontext.ReadTokenManagerFromContext(ctx)
	if tm == nil {
		log.Error(ctx, map[string]interface{}{
			"token": tm,
		}, "missing token manager")
		return false, errors.NewInternalError(ctx, errs.New("missing token manager"))
	}
	tokenWithClaims, err := jwt.ParseWithClaims(jwttoken.Raw, &auth.TokenPayload{}, func(t *jwt.Token) (interface{}, error) {
		return tm.(token.Manager).PublicKey(), nil
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err,
		}, "unable to parse the rpt token")
		return false, errors.NewInternalError(ctx, errs.Wrap(err, "unable to parse the rpt token"))
	}
	claims := tokenWithClaims.Claims.(*auth.TokenPayload)

	if claims.Authorization == nil {
		// No authorization in the token. This is not a RPT token. This is an access token.
		// We need to obtain an PRT token.
		log.Warn(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "no authorization found in the token; this is an access token (not a RPT token)")
		return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
	}

	// Check if the token was issued before the space resouces changed the last time.
	// If so, we need to re-fetch the rpt token for that space/resource and check permissions.
	outdated, err := s.isTokenOutdated(ctx, *claims, entitlementEndpoint, spaceID)
	if err != nil {
		return false, err
	}
	if outdated {
		return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
	}

	permissions := claims.Authorization.Permissions
	if permissions == nil {
		// if the RPT doesn't contain the resource info, it could be probably
		// because the entitlement was never fetched in the first place. Hence we consider
		// the token to be 'outdated' and hence re-fetch the entitlements from keycloak.
		return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
	}
	for _, permission := range permissions {
		name := permission.ResourceSetName
		if name != nil && spaceID == *name {
			return true, nil
		}
	}
	// if the RPT doesn't contain the resource info, it could be probably
	// because the entitlement was never fetched in the first place. Hence we consider
	// the token to be 'outdated' and hence re-fetch the entitlements from keycloak.
	return s.checkEntitlementForSpace(ctx, *jwttoken, entitlementEndpoint, spaceID)
}

func (s *KeycloakAuthzService) checkEntitlementForSpace(ctx context.Context, token jwt.Token, entitlementEndpoint string, spaceID string) (bool, error) {
	resource := auth.EntitlementResource{
		Permissions: []auth.ResourceSet{{Name: spaceID}},
	}
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &resource, token.Raw)
	if err != nil {
		return false, err
	}
	return ent != nil, nil
}

func (s *KeycloakAuthzService) isTokenOutdated(ctx context.Context, token auth.TokenPayload, entitlementEndpoint string, spaceID string) (bool, error) {
	spaceUUID, err := uuid.FromString(spaceID)
	if err != nil {
		return false, errors.NewInternalError(ctx, err)
	}
	var spaceResource *space.Resource
	err = application.Transactional(s.db, func(appl application.Application) error {
		spaceResource, err = appl.SpaceResources().LoadBySpace(ctx, &spaceUUID)
		return err
	})
	if err != nil {
		return false, err
	}
	if token.IssuedAt == 0 {
		return false, errors.NewInternalError(ctx, errs.New("iat claim is not found in the token"))
	}
	tokenIssued := time.Unix(token.IssuedAt, 0)
	return tokenIssued.Before(spaceResource.UpdatedAt), nil
}

// InjectAuthzService is a middleware responsible for setting up AuthzService in the context for every request.
func InjectAuthzService(service AuthzService) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx contx.Context, rw http.ResponseWriter, req *http.Request) error {
			config := service.Configuration()
			var endpoint string
			if config != nil {
				var err error
				endpoint, err = config.GetKeycloakEndpointEntitlement(&goa.RequestData{Request: req})
				if err != nil {
					log.Error(ctx, map[string]interface{}{
						"err": err,
					}, "unable to get entitlement endpoint")
					return err
				}
			}
			ctxWithAuthzServ := tokencontext.ContextWithSpaceAuthzService(ctx, &KeycloakAuthzServiceManager{Service: service, entitlementEndpoint: endpoint})
			return h(ctxWithAuthzServ, rw, req)
		}
	}
}

// Authorize returns true and the corresponding Requesting Party Token if the current user is among the space collaborators
func Authorize(ctx context.Context, spaceID string) (bool, error) {
	srv := tokencontext.ReadSpaceAuthzServiceFromContext(ctx)
	if srv == nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
		}, "Missing space authz service")

		return false, errs.New("missing space authz service")
	}
	manager := srv.(AuthzServiceManager)
	return manager.AuthzService().Authorize(ctx, manager.EntitlementEndpoint(), spaceID)
}
