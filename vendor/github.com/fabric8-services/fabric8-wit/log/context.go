package log

import (
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa/client"
	"github.com/goadesign/goa/middleware"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/pkg/errors"
)

// extractIdentityID obtains the identity ID out of the authentication context
func extractIdentityID(ctx context.Context) (string, error) {
	token := goajwt.ContextJWT(ctx)
	if token == nil {
		return "", errors.New("Missing token")
	}
	id := token.Claims.(jwt.MapClaims)["sub"]
	if id == nil {
		return "", errors.New("Missing sub")
	}

	return id.(string), nil
}

// extractRequestID obtains the request ID either from a goa client or middleware
func extractRequestID(ctx context.Context) string {
	reqID := middleware.ContextRequestID(ctx)
	if reqID == "" {
		return client.ContextRequestID(ctx)
	}

	return reqID
}
