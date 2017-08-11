package controller_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/resource"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/stretchr/testify/require"
)

const (
	// the various HTTP endpoints
	endpointWorkItem               = "/api/workitems"
	endpointWorkItems              = "/api/spaces/%s/workitems"
	endpointWorkItemTypes          = "/api/spaces/%s/workitemtypes"
	endpointWorkItemLinkCategories = "/api/workitemlinkcategories"
	endpointWorkItemLinkTypes      = "/api/spaces/%s/workitemlinktypes"
	endpointWorkItemLinks          = "/api/workitemlinks"

	endpointWorkItemRelationshipsLinks   = endpointWorkItem + "/%s/relationships/links"
	endpointWorkItemTypesSourceLinkTypes = endpointWorkItemTypes + "/%s/source-link-types"
	endpointWorkItemTypesTargetLinkTypes = endpointWorkItemTypes + "/%s/target-link-types"
)

// testSecureAPI defines how a Test object is.
type testSecureAPI struct {
	method             string
	url                string
	expectedStatusCode int    // this will be tested against responseRecorder.Code
	expectedErrorCode  string // this will be tested only if response body gets unmarshelled into app.JSONAPIErrors
	payload            *bytes.Buffer
	jwtToken           string
}

func (t testSecureAPI) String() string {
	return fmt.Sprintf(
		"TestSecureAPI { method: %v, url: %v, expectedStatusCode: %v, expectedErrorCode: %v }",
		t.method, t.url, t.expectedStatusCode, t.expectedStatusCode)
}

// getExpiredAuthHeader returns a JWT bearer token with an expiration date that lies in the past
func getExpiredAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{"exp": float64(time.Now().Unix() - 100)}
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

// getMalformedAuthHeader returns a JWT bearer token with the wrong prefix of "Malformed Bearer" instead of just "Bearer"
func getMalformedAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Malformed Bearer %s", tokenStr)
}

// getExpiredAuthHeader returns a valid JWT bearer token
func getValidAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

// UnauthorizeCreateUpdateDeleteTest will check authorized access to Create/Update/Delete APIs
func UnauthorizeCreateUpdateDeleteTest(t *testing.T, getDataFunc func(t *testing.T) []testSecureAPI, createServiceFunc func() *goa.Service, mountCtrlFunc func(service *goa.Service) error) {
	resource.Require(t, resource.Database)

	// This will be modified after merge PR for "Viper Environment configurations"
	publickey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(wittoken.RSAPublicKey))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	tokenTests := getDataFunc(t)

	for _, testObject := range tokenTests {
		// Build a request
		var req *http.Request
		var err error
		if testObject.payload == nil {
			req, err = http.NewRequest(testObject.method, testObject.url, nil)
		} else {
			req, err = http.NewRequest(testObject.method, testObject.url, testObject.payload)
		}
		// req, err := http.NewRequest(testObject.method, testObject.url, testObject.payload)
		if err != nil {
			t.Fatal("could not create a HTTP request")
		}
		// Add Authorization Header
		req.Header.Add("Authorization", testObject.jwtToken)

		rr := httptest.NewRecorder()

		service := createServiceFunc()
		require.NotNil(t, service)

		// if error is thrown during request processing, it will be caught by ErrorHandler middleware
		// this will put error code, status, details in recorder object.
		// e.g> {"id":"AL6spYb2","code":"jwt_security_error","status":401,"detail":"JWT validation failed: crypto/rsa: verification error"}
		//service.Use(middleware.ErrorHandler(service, true))
		// e.g. > {"errors":[{"code":"unknown_error","detail":"[19v4Bp8f] 401 jwt_security_error: JWT validation failed: Token is expired","status":"401","title":"Unauthorized"}]}
		service.Use(jsonapi.ErrorHandler(service, true))

		// append a middleware to service. Use appropriate RSA keys
		jwtMiddleware := goajwt.New(publickey, nil, app.NewJWTSecurity())
		// Adding middleware via "app" is important
		// Because it will check the design and accordingly apply the middleware if mentioned in design
		// But if I use `service.Use(jwtMiddleware)` then middleware is applied for all the requests (without checking design)
		app.UseJWTMiddleware(service, jwtMiddleware)

		if err := mountCtrlFunc(service); err != nil {
			t.Fatalf("Failed to mount controller: %s", err.Error())
		}

		// Hit the service with own request
		service.Mux.ServeHTTP(rr, req)

		require.Equal(t, testObject.expectedStatusCode, rr.Code, testObject.String())

		// Below code tries to open Body response which is expected to be a JSON
		// If could not parse it correctly into app.JSONAPIErrors
		// Then it gets logged and continue the test loop
		//fmt.Printf("\nrr.Body = %s\n", string(rr.Body.Bytes()))
		jerrors := app.JSONAPIErrors{}
		err = json.Unmarshal(rr.Body.Bytes(), &jerrors)
		if err != nil {
			t.Log("Could not parse JSON response: ", rr.Body)
			// safe to continue because we alread checked rr.Code=required_value
			continue
		}
		// Additional checks for 'more' confirmation
		require.Equal(t, testObject.expectedErrorCode, *jerrors.Errors[0].Code)
		require.Equal(t, strconv.Itoa(testObject.expectedStatusCode), *jerrors.Errors[0].Status, testObject.String())
	}
}

// The RSADifferentPrivateKeyTest key will be used to sign the token but verification should
// fail as this is not the key used by server security layer
// ssh-keygen -f test-wit
var RSADifferentPrivateKeyTest = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsIT76Mr3p8VvtSrzCVcXEcyvalUp50mm4yvfqvZ1fZfbqAzJ
c4GNJEkpBGoXF+WgjLNkPnwS+k1YuqvPeGG4vFPtErF7nxNCHpzU+cXScOOl3WrM
S5fj928sBSJiDIDBwc98mQbIKaCrpLSsFe/kapV5mHmmWGAx6dqnObbqtIte4M7w
arE/c8xW1Fww2YZ4e4Xwknm+Rs2kQmg0SJPpgRih05y3snEQjXz1kR9bGTEBUPmX
HBTySgA93gmimQUlSAT0+hz9hcYrwCgjbnXUHlcBP2VbB4omJ7L1zJc/XMPwINR/
PtkGRhL/DXXA4v/0MLkYDXXmZGku/X1+du2ypQIDAQABAoIBAFi0m3si9E2FNFvQ
l42sDFXPjJ9c6M/n/UvP8niRnf1dYO8Ube/zvJ/tfAVR4wUJSiMqy0dzRn4ufFZi
nMIcKZ/KdSqdskgAf4uuuIBEXzqHzAR29O9QBymC3pY97xPlaHki8bRc6h2xNlBw
0sG7agf90btD9soWnT6tuLeSKmRLh5aHUQv3aGwzPyNfKHQ8J/KwKdPudP+tVBsi
eNd7DZDgSEw6pRaSCKS3ChrsQQ2XPjGo+OI6HjZ/LAFhFXMq2cRGELGF766a6phK
aCTB619AXiRHdKE98zEY3GMDtXSgeA0yzxcbvr224rEkHGDfkZ0BJwOCqCiaw4tZ
F/lFDMkCgYEA36Uqyj0cML5rMwC/W4b6ihuK/DujBBFYPQ8eVYt5yUSyLNJn5RLt
33eBUvgGB/FyAio5aCp49mcPtfFv5GKXpzTSYo/bWV1iy+oVwgPF7UA5gvtRw90w
NScLNsZ/7fOEpPJvlsKq/PQoMIoAjkegoj95cqM1yzC3aZpaAjx6188CgYEAyg58
5e5WK3zXICMpE8q+1AB+kJ/3UhQ71kpK4Xml0PtTJ7Bzqn0hiU4ThfpKj1n9PtpU
9Op3PqcfVjf11SA1tI5LRHQvgUSNppvf2hPgW8QrqRs5YFgNg0DkVXs3OxWIA4QA
Ko6oZu2ZpvK3TdYXRmcRUXXNyCDoSmJvH339N0sCgYB0g1kCmcm4/0tb+/S1m2Gl
V+oVtIAeG2csEFdOW+ar27Uzsr5b0nvI4zql3f9OXhR2WkckJJR2UoUV1d3kTxUR
EGzW2nl9WjChaafCNzMDgmUz/vi/INn/pwKpm8qETkz5njBSi8KHHDBf8VWOynQ+
cvEzryHUZOH5C2f/KEEbcwKBgQCGzVGgaPjOPJSdWTfPf4T+lXHa9Q4gkWU2WwxI
D0uD+BiLMxqH1MGqBA/cY5aYutXMuAbT+xUhFIhAkkcNMFcEJaarfcQvvteuHvIi
YP5e2qqyQHpv/27McV+kc/buEThT+B3QRqqtOLk4+1c1s66Fhr+0FB789I9lCPTQ
EtL7rwKBgQC5x7lGs+908uqf7yFXHzw7rPGFUe6cuxZ3jVOzovVoXRma+C7nroNx
/A4rWPqfpmiKcmrd7K4DQFlYhoq+MALEDmQm+/8G6j2inF53fRGgJVzaZhSvnO9X
CMnDipW5SU9AQE+xC8Zc+02rcyuZ7ha1WXKgIKwAa92jmJSCJjzdxA==
-----END RSA PRIVATE KEY-----`
