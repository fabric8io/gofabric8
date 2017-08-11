package token

import (
	"crypto/rsa"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// GenerateToken generates a JWT token and signs it using the given private key
func GenerateToken(identityID string, identityUsername string, privateKey *rsa.PrivateKey) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims.(jwt.MapClaims)["uuid"] = identityID
	token.Claims.(jwt.MapClaims)["preferred_username"] = identityUsername
	token.Claims.(jwt.MapClaims)["sub"] = identityID

	tokenStr, err := token.SignedString(privateKey)
	if err != nil {
		return "", errors.WithStack(err)
	}
	return tokenStr, nil
}
