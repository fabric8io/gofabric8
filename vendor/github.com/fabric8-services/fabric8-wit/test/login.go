package test

import (
	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/auth"
	tokencontext "github.com/fabric8-services/fabric8-wit/login/tokencontext"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/token"

	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

type dummySpaceAuthzService struct {
}

func (s *dummySpaceAuthzService) Authorize(ctx context.Context, entitlementEndpoint string, spaceID string) (bool, error) {
	return true, nil
}

func (s *dummySpaceAuthzService) Configuration() authz.AuthzConfiguration {
	return nil
}

// WithIdentity fills the context with token
// Token is filled using input Identity object
func WithIdentity(ctx context.Context, ident account.Identity) context.Context {
	token := fillClaimsWithIdentity(ident)
	return goajwt.WithJWT(ctx, token)
}

// WithAuthz fills the context with token
// Token is filled using input Identity object and resource authorization information
func WithAuthz(ctx context.Context, key interface{}, ident account.Identity, authz auth.AuthorizationPayload) context.Context {
	token := fillClaimsWithIdentity(ident)
	token.Claims.(jwt.MapClaims)["authorization"] = authz
	t, err := token.SignedString(key)
	if err != nil {
		panic(err.Error())
	}
	token.Raw = t
	return goajwt.WithJWT(ctx, token)
}

func fillClaimsWithIdentity(ident account.Identity) *jwt.Token {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["sub"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["uuid"] = ident.ID.String()
	token.Claims.(jwt.MapClaims)["fullName"] = ident.User.FullName
	token.Claims.(jwt.MapClaims)["imageURL"] = ident.User.ImageURL
	token.Claims.(jwt.MapClaims)["iat"] = time.Now().Unix()
	return token
}

func service(serviceName string, tm token.Manager, key interface{}, u account.Identity, authz *auth.AuthorizationPayload) *goa.Service {
	svc := goa.New(serviceName)
	if authz == nil {
		svc.Context = WithIdentity(svc.Context, u)
	} else {
		svc.Context = WithAuthz(svc.Context, key, u, *authz)
	}
	svc.Context = tokencontext.ContextWithTokenManager(svc.Context, tm)
	return svc
}

// ServiceAsUserWithAuthz creates a new service and fill the context with input Identity and resource authorization information
func ServiceAsUserWithAuthz(serviceName string, tm token.Manager, key interface{}, u account.Identity, authorizationPayload auth.AuthorizationPayload) *goa.Service {
	svc := service(serviceName, tm, key, u, &authorizationPayload)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: &dummySpaceAuthzService{}})
	return svc
}

// ServiceAsUser creates a new service and fill the context with input Identity
func ServiceAsUser(serviceName string, tm token.Manager, u account.Identity) *goa.Service {
	svc := service(serviceName, tm, nil, u, nil)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: &dummySpaceAuthzService{}})
	return svc
}

// ServiceAsSpaceUser creates a new service and fill the context with input Identity and space authz service
func ServiceAsSpaceUser(serviceName string, tm token.Manager, u account.Identity, authzSrv authz.AuthzService) *goa.Service {
	svc := service(serviceName, tm, nil, u, nil)
	svc.Context = tokencontext.ContextWithSpaceAuthzService(svc.Context, &authz.KeycloakAuthzServiceManager{Service: authzSrv})
	return svc
}
