package token_test

import (
	"testing"
	"time"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/resource"
	testtoken "github.com/fabric8-services/fabric8-wit/test/token"
	"github.com/fabric8-services/fabric8-wit/token"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestExtractToken(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	manager := createManager(t)

	identity := account.Identity{
		ID:       uuid.NewV4(),
		Username: "testuser",
	}
	privateKey, err := token.ParsePrivateKey([]byte(token.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse private key", err)
	}

	token, err := testtoken.GenerateToken(identity.ID.String(), identity.Username, privateKey)
	if err != nil {
		t.Fatal("Could not generate test token", err)
	}

	ident, err := manager.Extract(token)
	if err != nil {
		t.Fatal("Could not extract Identity from generated token", err)
	}
	assert.Equal(t, identity.Username, ident.Username)
}

func TestExtractWithInvalidToken(t *testing.T) {
	// This tests generates invalid Token
	// by setting expired date, empty UUID, not setting UUID
	// all above cases are invalid
	// hence manager.Extract should fail in all above cases
	manager := createManager(t)
	privateKey, err := token.ParsePrivateKey([]byte(token.RSAPrivateKey))

	tok := jwt.New(jwt.SigningMethodRS256)
	// add already expired time to "exp" claim"
	claims := jwt.MapClaims{"sub": "some_uuid", "exp": float64(time.Now().Unix() - 100)}
	tok.Claims = claims
	tokenStr, err := tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err := manager.Extract(tokenStr)
	if err == nil {
		t.Error("Expired token should not be parsed. Error must not be nil", idn, err)
	}

	// now set correct EXP but do not set uuid
	claims = jwt.MapClaims{"exp": float64(time.Now().AddDate(0, 0, 1).Unix())}
	tok.Claims = claims
	tokenStr, err = tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err = manager.Extract(tokenStr)
	if err == nil {
		t.Error("Invalid token should not be parsed. Error must not be nil", idn, err)
	}

	// now set UUID to empty String
	claims = jwt.MapClaims{"sub": ""}
	tok.Claims = claims
	tokenStr, err = tok.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	idn, err = manager.Extract(tokenStr)
	if err == nil {
		t.Error("Invalid token should not be parsed. Error must not be nil", idn, err)
	}
}

func TestLocateTokenInContex(t *testing.T) {
	id := uuid.NewV4()

	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = id.String()
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	foundId, err := manager.Locate(ctx)
	if err != nil {
		t.Error("Failed not locate token in given context", err)
	}
	assert.Equal(t, id, foundId, "ID in created context not equal")
}

func TestLocateMissingTokenInContext(t *testing.T) {
	ctx := context.Background()

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateMissingUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func TestLocateInvalidUUIDInTokenInContext(t *testing.T) {
	tk := jwt.New(jwt.SigningMethodRS256)
	tk.Claims.(jwt.MapClaims)["sub"] = "131"
	ctx := goajwt.WithJWT(context.Background(), tk)

	manager := createManager(t)

	_, err := manager.Locate(ctx)
	if err == nil {
		t.Error("Should have returned error on missing token in contex", err)
	}
}

func createManager(t *testing.T) token.Manager {
	privateKey, err := token.ParsePrivateKey([]byte(token.RSAPrivateKey))
	if err != nil {
		t.Fatal("Could not parse private key")
	}

	return token.NewManagerWithPrivateKey(privateKey)
}
