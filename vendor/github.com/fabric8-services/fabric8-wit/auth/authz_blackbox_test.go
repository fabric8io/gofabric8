package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/auth"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"crypto/rsa"

	_ "github.com/lib/pq"
)

var (
	configuration *config.ConfigurationData
	scopes        = []string{"read:test", "admin:test"}
	publicKey     *rsa.PublicKey
)

func init() {
	var err error
	configuration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	publicKey, err = wittoken.ParsePublicKey([]byte(configuration.GetTokenPublicKey()))
	if err != nil {
		panic(fmt.Errorf("Failed to parse the public key: %s", err.Error()))
	}
}

func TestAuth(t *testing.T) {
	resource.Require(t, resource.Remote)
	suite.Run(t, new(TestAuthSuite))
}

type TestAuthSuite struct {
	suite.Suite
}

func (s *TestAuthSuite) SetupSuite() {
}

func (s *TestAuthSuite) TearDownSuite() {
	cleanKeycloakResources(s.T())
}

func (s *TestAuthSuite) TestCreateAndDeleteResourceOK() {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	ctx := context.Background()
	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(s.T(), err)
	pat := getProtectedAPITokenOK(s.T())

	id, _ := createResource(s.T(), ctx, pat)
	deleteResource(s.T(), ctx, id, authzEndpoint, pat)
}

func (s *TestAuthSuite) TestDeleteNonexistingResourceFails() {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}

	ctx := context.Background()

	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(s.T(), err)
	pat := getProtectedAPITokenOK(s.T())
	err = auth.DeleteResource(ctx, uuid.NewV4().String(), authzEndpoint, pat)
	require.NotNil(s.T(), err)
}

func (s *TestAuthSuite) TestCreatePolicyOK() {
	ctx := context.Background()
	pat := getProtectedAPITokenOK(s.T())
	clientId, clientsEndpoint := getClientIDAndEndpoint(s.T())

	id, policy := createPolicy(s.T(), ctx, pat)
	defer deletePolicy(s.T(), ctx, clientsEndpoint, clientId, id, pat)

	pl := validatePolicy(s.T(), ctx, clientsEndpoint, clientId, policy, id, pat)

	firstTestUserID := getUserID(s.T(), configuration.GetKeycloakTestUserName(), configuration.GetKeycloakTestUserSecret())
	pl.Config = auth.PolicyConfigData{
		UserIDs: "[\"" + firstTestUserID + "\"]",
	}
	err := auth.UpdatePolicy(ctx, clientsEndpoint, clientId, *pl, pat)
	require.Nil(s.T(), err)
	validatePolicy(s.T(), ctx, clientsEndpoint, clientId, *pl, id, pat)
}

func (s *TestAuthSuite) TestDeletePolicyOK() {
	ctx := context.Background()
	pat := getProtectedAPITokenOK(s.T())
	clientId, clientsEndpoint := getClientIDAndEndpoint(s.T())

	id, _ := createPolicy(s.T(), ctx, pat)
	deletePolicy(s.T(), ctx, clientsEndpoint, clientId, id, pat)

	_, err := auth.GetPolicy(ctx, clientsEndpoint, clientId, id, pat)
	require.NotNil(s.T(), err)
	require.IsType(s.T(), errors.NotFoundError{}, err)
}

func (s *TestAuthSuite) TestCreateAndDeletePermissionOK() {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(s.T(), err)

	ctx := context.Background()
	pat := getProtectedAPITokenOK(s.T())

	resourceID, _ := createResource(s.T(), ctx, pat)
	defer deleteResource(s.T(), ctx, resourceID, authzEndpoint, pat)
	clientId, clientsEndpoint := getClientIDAndEndpoint(s.T())
	policyID, _ := createPolicy(s.T(), ctx, pat)
	defer deletePolicy(s.T(), ctx, clientsEndpoint, clientId, policyID, pat)

	permission := auth.KeycloakPermission{
		Name:             "test-" + uuid.NewV4().String(),
		Type:             auth.PermissionTypeResource,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
		// "config":{"resources":"[\"<ResourceID>\"]","applyPolicies":"[\"<PolicyID>\"]"}
		Config: auth.PermissionConfigData{
			Resources:     "[\"" + resourceID + "\"]",
			ApplyPolicies: "[\"" + policyID + "\"]",
		},
	}

	id, err := auth.CreatePermission(ctx, clientsEndpoint, clientId, permission, pat)
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), "", id)
	deletePermission(s.T(), ctx, clientsEndpoint, clientId, id, pat)
}

func (s *TestAuthSuite) TestDeleteNonexistingPolicyAndPermissionFails() {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}

	ctx := context.Background()

	clientsEndpoint, err := configuration.GetKeycloakEndpointClients(r)
	require.Nil(s.T(), err)
	pat := getProtectedAPITokenOK(s.T())
	clientId, _ := getClientIDAndEndpoint(s.T())
	err = auth.DeletePolicy(ctx, clientsEndpoint, clientId, uuid.NewV4().String(), pat)
	assert.NotNil(s.T(), err)

	err = auth.DeletePermission(ctx, clientsEndpoint, clientId, uuid.NewV4().String(), pat)
	assert.NotNil(s.T(), err)
}

func (s *TestAuthSuite) TestGetEntitlement() {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(s.T(), err)

	ctx := context.Background()
	pat := getProtectedAPITokenOK(s.T())

	resourceID, resourceName := createResource(s.T(), ctx, pat)
	defer deleteResource(s.T(), ctx, resourceID, authzEndpoint, pat)
	clientId, clientsEndpoint := getClientIDAndEndpoint(s.T())
	policyID, _ := createPolicy(s.T(), ctx, pat)
	defer deletePolicy(s.T(), ctx, clientsEndpoint, clientId, policyID, pat)

	permission := auth.KeycloakPermission{
		Name:             "test-" + uuid.NewV4().String(),
		Type:             auth.PermissionTypeResource,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
		// "config":{"resources":"[\"<ResourceID>\"]","applyPolicies":"[\"<PolicyID>\"]"}
		Config: auth.PermissionConfigData{
			Resources:     "[\"" + resourceID + "\"]",
			ApplyPolicies: "[\"" + policyID + "\"]",
		},
	}

	permissionID, err := auth.CreatePermission(ctx, clientsEndpoint, clientId, permission, pat)
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), "", permissionID)
	defer deletePermission(s.T(), ctx, clientsEndpoint, clientId, permissionID, pat)

	entitlementEndpoint, err := configuration.GetKeycloakEndpointEntitlement(r)
	require.Nil(s.T(), err)
	tokenEndpoint, err := configuration.GetKeycloakEndpointToken(r)
	require.Nil(s.T(), err)
	testUserToken, err := controller.GenerateUserToken(ctx, tokenEndpoint, configuration, configuration.GetKeycloakTestUserName(), configuration.GetKeycloakTestUserSecret())
	// {"permissions" : [{"resource_set_name" : "<spaceID>"}]}
	entitlementResource := auth.EntitlementResource{
		Permissions: []auth.ResourceSet{{Name: resourceName}},
	}
	ent, err := auth.GetEntitlement(ctx, entitlementEndpoint, &entitlementResource, *testUserToken.Token.AccessToken)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), ent)
	require.NotEqual(s.T(), "", ent)

	ok, err := auth.VerifyResourceUser(ctx, *testUserToken.Token.AccessToken, resourceName, entitlementEndpoint)
	require.True(s.T(), ok)
	require.Nil(s.T(), err)

	secondTestUserID := getUserID(s.T(), configuration.GetKeycloakTestUser2Name(), configuration.GetKeycloakTestUser2Secret())
	pl, err := auth.GetPolicy(ctx, clientsEndpoint, clientId, policyID, pat)
	pl.Config = auth.PolicyConfigData{
		UserIDs: "[\"" + secondTestUserID + "\"]",
	}
	err = auth.UpdatePolicy(ctx, clientsEndpoint, clientId, *pl, pat)
	require.Nil(s.T(), err)

	ent, err = auth.GetEntitlement(ctx, entitlementEndpoint, &entitlementResource, *testUserToken.Token.AccessToken)
	require.Nil(s.T(), err)
	require.Nil(s.T(), ent)

	ok, err = auth.VerifyResourceUser(ctx, *testUserToken.Token.AccessToken, resourceName, entitlementEndpoint)
	require.False(s.T(), ok)
	require.Nil(s.T(), err)

	ent, err = auth.GetEntitlement(ctx, entitlementEndpoint, nil, *testUserToken.Token.AccessToken)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), ent)
	require.NotEqual(s.T(), "", ent)

	if int64(len(*ent)) > configuration.GetHeaderMaxLength() {
		// The RPT token is too long. Remove existing resources and re-obtain the entitlement
		require.Nil(s.T(), CleanupResources(s.T(), ctx, *ent, authzEndpoint, pat, resourceID))

		ent, err = auth.GetEntitlement(ctx, entitlementEndpoint, nil, *testUserToken.Token.AccessToken)
		require.Nil(s.T(), err)
		require.NotNil(s.T(), ent)
		require.NotEqual(s.T(), "", ent)
	}

	ent, err = auth.GetEntitlement(ctx, entitlementEndpoint, nil, *ent)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), ent)
	require.NotEqual(s.T(), "", ent)
}

func CleanupResources(t *testing.T, ctx context.Context, rpt string, authzEndpoint string, pat string, excludeResourceID string) error {
	tokenWithClaims, err := jwt.ParseWithClaims(rpt, &auth.TokenPayload{}, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	if err != nil {
		return err
	}
	claims := tokenWithClaims.Claims.(*auth.TokenPayload)
	permissions := claims.Authorization.Permissions
	if permissions == nil {
		return nil
	}
	clientId, clientsEndpoint := getClientIDAndEndpoint(t)
	for _, permission := range permissions {
		if excludeResourceID != *permission.ResourceSetID {
			policyEndpoint := fmt.Sprintf("%s/%s/authz/resource-server/policy?first=0&max=100&resource=%s", clientsEndpoint, clientId, *permission.ResourceSetID)
			req, err := http.NewRequest("GET", policyEndpoint, nil)
			require.Nil(t, err)
			req.Header.Add("Authorization", "Bearer "+pat)
			res, err := http.DefaultClient.Do(req)
			require.Nil(t, err)
			require.Equal(t, 200, res.StatusCode)

			jsonString := rest.ReadBody(res.Body)
			var policyResult []policyRequestResultPayload
			err = json.Unmarshal([]byte(jsonString), &policyResult)
			require.Nil(t, err)
			for _, policy := range policyResult {
				deletePolicy(t, ctx, clientsEndpoint, clientId, policy.ID, pat)
			}

			deleteResource(t, ctx, *permission.ResourceSetID, authzEndpoint, pat)
		}
	}
	return nil
}

func (s *TestAuthSuite) TestGetClientIDOK() {
	id, _ := getClientIDAndEndpoint(s.T())
	assert.Equal(s.T(), "239ed057-eec1-425b-a7eb-f4b338c94cdd", id)
}

func (s *TestAuthSuite) TestGetProtectedAPITokenOK() {
	token := getProtectedAPITokenOK(s.T())
	require.NotEqual(s.T(), "", token)
}

func (s *TestAuthSuite) TestReadTokenOK() {
	b := closer{bytes.NewBufferString("{\"access_token\":\"accToken\", \"expires_in\":3000000, \"refresh_expires_in\":2, \"refresh_token\":\"refToken\"}")}
	response := http.Response{Body: b}
	token, err := auth.ReadToken(context.Background(), &response)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "accToken", *token.AccessToken)
	assert.Equal(s.T(), int64(3000000), *token.ExpiresIn)
	assert.Equal(s.T(), int64(2), *token.RefreshExpiresIn)
	assert.Equal(s.T(), "refToken", *token.RefreshToken)
}

func (s *TestAuthSuite) TestUpdateUserToPolicyOK() {
	policy := auth.KeycloakPolicy{
		Name:             "test-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	userID1 := uuid.NewV4().String()
	userID2 := uuid.NewV4().String()
	userID3 := uuid.NewV4().String()
	assert.True(s.T(), policy.AddUserToPolicy(userID1))
	//"users":"[\"<ID>\",\"<ID>\"]"
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\"]", userID1), policy.Config.UserIDs)
	assert.True(s.T(), policy.AddUserToPolicy(userID2))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\"]", userID1, userID2), policy.Config.UserIDs)
	assert.False(s.T(), policy.AddUserToPolicy(userID2))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\"]", userID1, userID2), policy.Config.UserIDs)
	assert.True(s.T(), policy.AddUserToPolicy(userID3))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\",\"%s\"]", userID1, userID2, userID3), policy.Config.UserIDs)
	assert.False(s.T(), policy.RemoveUserFromPolicy(uuid.NewV4().String()))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\",\"%s\"]", userID1, userID2, userID3), policy.Config.UserIDs)
	assert.True(s.T(), policy.RemoveUserFromPolicy(userID2))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\"]", userID1, userID3), policy.Config.UserIDs)
	assert.True(s.T(), policy.RemoveUserFromPolicy(userID1))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\"]", userID3), policy.Config.UserIDs)
	assert.True(s.T(), policy.AddUserToPolicy(userID2))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\",\"%s\"]", userID3, userID2), policy.Config.UserIDs)
	assert.True(s.T(), policy.RemoveUserFromPolicy(userID3))
	assert.Equal(s.T(), fmt.Sprintf("[\"%s\"]", userID2), policy.Config.UserIDs)
	assert.True(s.T(), policy.RemoveUserFromPolicy(userID2))
	assert.Equal(s.T(), "[]", policy.Config.UserIDs)
}

func deleteResource(t *testing.T, ctx context.Context, id string, authzEndpoint string, pat string) {
	err := auth.DeleteResource(ctx, id, authzEndpoint, pat)
	assert.Nil(t, err)
}

type resourceRequestResultPayload struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
	ID   string `json:"_id"`
}

type policyRequestResultPayload struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func cleanKeycloakResources(t *testing.T) {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	ctx := context.Background()
	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(t, err)

	clientId, clientsEndpoint := getClientIDAndEndpoint(t)
	resourceEndpoint := clientsEndpoint + "/" + clientId + "/authz/resource-server/resource?deep=false&first=0&max=1000&name=test"
	pat := getProtectedAPITokenOK(t)

	req, err := http.NewRequest("GET", resourceEndpoint, nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", "Bearer "+pat)
	res, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, res.StatusCode)

	jsonString := rest.ReadBody(res.Body)

	var result []resourceRequestResultPayload
	err = json.Unmarshal([]byte(jsonString), &result)
	require.Nil(t, err)
	for _, res := range result {
		deleteResource(t, ctx, res.ID, authzEndpoint, pat)
	}

	policyEndpoint := clientsEndpoint + "/" + clientId + "/authz/resource-server/policy?first=0&max=1000&name=test&permission=false"
	req, err = http.NewRequest("GET", policyEndpoint, nil)
	require.Nil(t, err)
	req.Header.Add("Authorization", "Bearer "+pat)
	res, err = http.DefaultClient.Do(req)
	require.Nil(t, err)
	require.Equal(t, 200, res.StatusCode)

	jsonString = rest.ReadBody(res.Body)
	var policyResult []policyRequestResultPayload
	err = json.Unmarshal([]byte(jsonString), &policyResult)
	require.Nil(t, err)
	for _, policy := range policyResult {
		deletePolicy(t, ctx, clientsEndpoint, clientId, policy.ID, pat)
	}
}

func createResource(t *testing.T, ctx context.Context, pat string) (string, string) {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	uri := "testResourceURI"
	kcResource := auth.KeycloakResource{
		Name:   "test-" + uuid.NewV4().String(),
		Type:   "testResource",
		URI:    &uri,
		Scopes: &scopes,
	}
	authzEndpoint, err := configuration.GetKeycloakEndpointAuthzResourceset(r)
	require.Nil(t, err)

	id, err := auth.CreateResource(ctx, kcResource, authzEndpoint, pat)
	require.Nil(t, err)
	require.NotEqual(t, "", id)
	return id, kcResource.Name
}

func createPolicy(t *testing.T, ctx context.Context, pat string) (string, auth.KeycloakPolicy) {
	firstTestUserID := getUserID(t, configuration.GetKeycloakTestUserName(), configuration.GetKeycloakTestUserSecret())
	secondTestUserID := getUserID(t, configuration.GetKeycloakTestUser2Name(), configuration.GetKeycloakTestUser2Secret())
	policy := auth.KeycloakPolicy{
		Name:             "test-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	assert.True(t, policy.AddUserToPolicy(firstTestUserID))
	assert.True(t, policy.AddUserToPolicy(secondTestUserID))

	clientId, clientsEndpoint := getClientIDAndEndpoint(t)

	id, err := auth.CreatePolicy(ctx, clientsEndpoint, clientId, policy, pat)
	require.Nil(t, err)
	require.NotEqual(t, "", id)
	return id, policy
}

func deletePolicy(t *testing.T, ctx context.Context, clientsEndpoint string, clientId string, id string, pat string) {
	err := auth.DeletePolicy(ctx, clientsEndpoint, clientId, id, pat)
	assert.Nil(t, err)
}

func deletePermission(t *testing.T, ctx context.Context, clientsEndpoint string, clientId string, id string, pat string) {
	err := auth.DeletePermission(ctx, clientsEndpoint, clientId, id, pat)
	assert.Nil(t, err)
}

func validatePolicy(t *testing.T, ctx context.Context, clientsEndpoint string, clientId string, policyToValidate auth.KeycloakPolicy, remotePolicyId string, pat string) *auth.KeycloakPolicy {
	pl, err := auth.GetPolicy(ctx, clientsEndpoint, clientId, remotePolicyId, pat)
	assert.Nil(t, err)
	assert.Equal(t, policyToValidate.Name, pl.Name)
	assert.Equal(t, policyToValidate.Type, pl.Type)
	assert.Equal(t, policyToValidate.Logic, pl.Logic)
	assert.Equal(t, policyToValidate.Type, pl.Type)
	assert.Equal(t, policyToValidate.DecisionStrategy, pl.DecisionStrategy)
	assert.Equal(t, policyToValidate.Config.UserIDs, pl.Config.UserIDs)
	return pl
}

func getUserID(t *testing.T, username string, usersecret string) string {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}

	tokenEndpoint, err := configuration.GetKeycloakEndpointToken(r)
	require.Nil(t, err)
	userinfoEndpoint, err := configuration.GetKeycloakEndpointUserInfo(r)
	require.Nil(t, err)
	adminEndpoint, err := configuration.GetKeycloakEndpointAdmin(r)
	require.Nil(t, err)

	ctx := context.Background()
	testToken, err := controller.GenerateUserToken(ctx, tokenEndpoint, configuration, username, usersecret)
	require.Nil(t, err)
	accessToken := testToken.Token.AccessToken
	require.NotNil(t, accessToken)
	userinfo, err := auth.GetUserInfo(ctx, userinfoEndpoint, *accessToken)
	require.Nil(t, err)
	userID := userinfo.Sub
	pat := getProtectedAPITokenOK(t)
	ok, err := auth.ValidateKeycloakUser(ctx, adminEndpoint, userID, pat)
	require.Nil(t, err)
	require.True(t, ok)
	return userID
}

func getClientIDAndEndpoint(t *testing.T) (string, string) {
	r := &goa.RequestData{
		Request: &http.Request{Host: "domain.io"},
	}
	clientsEndpoint, err := configuration.GetKeycloakEndpointClients(r)
	require.Nil(t, err)
	publicClientID := configuration.GetKeycloakClientID()
	require.Nil(t, err)
	pat := getProtectedAPITokenOK(t)

	id, err := auth.GetClientID(context.Background(), clientsEndpoint, publicClientID, pat)
	require.Nil(t, err)
	return id, clientsEndpoint
}

func getProtectedAPITokenOK(t *testing.T) string {
	r := &goa.RequestData{
		Request: &http.Request{Host: "demo.api.openshift.io"},
	}

	endpoint, err := configuration.GetKeycloakEndpointToken(r)
	require.Nil(t, err)
	token, err := auth.GetProtectedAPIToken(context.Background(), endpoint, configuration.GetKeycloakClientID(), configuration.GetKeycloakSecret())
	require.Nil(t, err)
	return token
}

type closer struct {
	io.Reader
}

func (closer) Close() error {
	return nil
}
