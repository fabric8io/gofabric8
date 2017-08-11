package goamiddleware

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

// TokenContext is a new goa middleware that aims to extract the token from the
// Authorization header when possible. If the Authorization header is missing in the request,
// no error is returned. However, if the Authorization header contains a
// token, it will be stored it in the context.
func TokenContext(validationKeys interface{}, validationFunc goa.Middleware, scheme *goa.JWTSecurity) goa.Middleware {
	var rsaKeys []*rsa.PublicKey
	var hmacKeys [][]byte

	rsaKeys, ecdsaKeys, hmacKeys := partitionKeys(validationKeys)

	return func(nextHandler goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			// TODO: implement the QUERY string handler too
			if scheme.In != goa.LocHeader {
				log.Error(ctx, nil, fmt.Sprintf("whoops, security scheme with location (in) %q not supported", scheme.In))
				return fmt.Errorf("whoops, security scheme with location (in) %q not supported", scheme.In)
			}
			val := req.Header.Get(scheme.Name)
			if val != "" && strings.HasPrefix(strings.ToLower(val), "bearer ") {
				log.Debug(ctx, nil, "found header 'Authorization: Bearer JWT-token...'")
				incomingToken := strings.Split(val, " ")[1]
				log.Debug(ctx, nil, "extracted the incoming token %v ", incomingToken)

				var (
					token  *jwt.Token
					err    error
					parsed = false
				)

				if len(rsaKeys) > 0 {
					token, err = parseRSAKeys(rsaKeys, "RS", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed && len(ecdsaKeys) > 0 {
					token, err = parseECDSAKeys(ecdsaKeys, "ES", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed && len(hmacKeys) > 0 {
					token, err = parseHMACKeys(hmacKeys, "HS", incomingToken)
					if err == nil {
						parsed = true
					}
				}

				if !parsed {
					log.Warn(ctx, nil, "unable to parse JWT token: %v", err)
				}

				ctx = goajwt.WithJWT(ctx, token)
			}

			return nextHandler(ctx, rw, req)
		}
	}
}

// partitionKeys sorts keys by their type.
func partitionKeys(k interface{}) ([]*rsa.PublicKey, []*ecdsa.PublicKey, [][]byte) {
	var (
		rsaKeys   []*rsa.PublicKey
		ecdsaKeys []*ecdsa.PublicKey
		hmacKeys  [][]byte
	)

	switch typed := k.(type) {
	case []byte:
		hmacKeys = append(hmacKeys, typed)
	case [][]byte:
		hmacKeys = typed
	case string:
		hmacKeys = append(hmacKeys, []byte(typed))
	case []string:
		for _, s := range typed {
			hmacKeys = append(hmacKeys, []byte(s))
		}
	case *rsa.PublicKey:
		rsaKeys = append(rsaKeys, typed)
	case []*rsa.PublicKey:
		rsaKeys = typed
	case *ecdsa.PublicKey:
		ecdsaKeys = append(ecdsaKeys, typed)
	case []*ecdsa.PublicKey:
		ecdsaKeys = typed
	}

	return rsaKeys, ecdsaKeys, hmacKeys
}

func parseRSAKeys(rsaKeys []*rsa.PublicKey, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, pubkey := range rsaKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]))
			}
			return pubkey, nil
		})
		if err == nil {
			return
		}
	}
	return
}

func parseECDSAKeys(ecdsaKeys []*ecdsa.PublicKey, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, pubkey := range ecdsaKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]))
			}
			return pubkey, nil
		})
		if err == nil {
			return
		}
	}
	return
}

func parseHMACKeys(hmacKeys [][]byte, algo, incomingToken string) (token *jwt.Token, err error) {
	for _, key := range hmacKeys {
		token, err = jwt.Parse(incomingToken, func(token *jwt.Token) (interface{}, error) {
			if !strings.HasPrefix(token.Method.Alg(), algo) {
				return nil, goajwt.ErrJWTError(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]))
			}
			return key, nil
		})
		if err == nil {
			return
		}
	}
	return
}
