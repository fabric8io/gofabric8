package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

/*  For recent spaces test */

type TestRecentSpacesREST struct {
	gormtestsupport.RemoteTestSuite
	configuration      *configuration.ConfigurationData
	tokenManager       token.Manager
	identityRepository *MockIdentityRepository
	userRepository     *MockUserRepository
	loginService       *TestLoginService

	clean func()
}

func TestRunRecentSpacesREST(t *testing.T) {
	suite.Run(t, &TestRecentSpacesREST{RemoteTestSuite: gormtestsupport.NewRemoteTestSuite("../config.yaml")})
}

func (rest *TestRecentSpacesREST) newTestKeycloakOAuthProvider(db application.DB) *login.KeycloakOAuthProvider {
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	require.Nil(rest.T(), err)
	tokenManager := token.NewManager(publicKey)

	return login.NewKeycloakOAuthProvider(rest.identityRepository, rest.userRepository, tokenManager, db)
}

func (rest *TestRecentSpacesREST) SetupTest() {
	c, err := configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	rest.configuration = c
	publicKey, err := token.ParsePublicKey([]byte(rest.configuration.GetTokenPublicKey()))
	require.Nil(rest.T(), err)
	rest.tokenManager = token.NewManager(publicKey)

	identity := account.Identity{}
	user := account.User{}
	identity.User = user

	rest.loginService = &TestLoginService{}

	rest.identityRepository = &MockIdentityRepository{testIdentity: &identity}
	rest.userRepository = &MockUserRepository{}

}

/* MockUserRepositoryService */

type MockIdentityRepository struct {
	testIdentity *account.Identity
}

func (rest *TestRecentSpacesREST) SecuredController() (*goa.Service, *LoginController) {
	svc := testsupport.ServiceAsUser("Login-Service", rest.tokenManager, testsupport.TestIdentity)
	loginController := &LoginController{
		Controller:         svc.NewController("login"),
		auth:               rest.loginService,
		tokenManager:       rest.tokenManager,
		configuration:      rest.configuration,
		identityRepository: rest.identityRepository,
	}
	return svc, loginController
}

func (rest *TestRecentSpacesREST) TestResourceRequestPayload() {
	t := rest.T()
	resource.Require(t, resource.Remote)
	service, controller := rest.SecuredController()

	// Generate an access token for a test identity
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	tokenEndpoint, err := rest.configuration.GetKeycloakEndpointToken(r)
	require.Nil(t, err)

	accessToken, err := GenerateUserToken(service.Context, tokenEndpoint, rest.configuration, rest.configuration.GetKeycloakTestUserName(), rest.configuration.GetKeycloakTestUserSecret())
	require.Nil(t, err)

	accessTokenString := accessToken.Token.AccessToken

	require.Nil(t, err)
	require.NotNil(t, accessTokenString)

	require.Nil(t, err)

	// Scenario 1 - Test user has a nil contextInformation, hence there are no recent spaces to
	// add to the resource object

	rest.identityRepository.testIdentity.User.ContextInformation = nil
	resource, err := controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	// This will be nil because contextInformation for the test user is empty!
	require.Nil(t, resource)

	// Scenario 2 - Test user has 'some' contextInformation incl. 12 recent spaces.
	identity := account.Identity{}
	dummyRecentSpaces := []interface{}{}
	for i := 1; i <= maxRecentSpacesForRPT+2; i++ {
		dummyRecentSpaces = append(dummyRecentSpaces, uuid.NewV4().String())
	}
	user := account.User{
		ContextInformation: account.ContextInformation{
			"recentSpaces": dummyRecentSpaces,
		},
	}
	identity.User = user
	rest.identityRepository.testIdentity = &identity

	//Use the same access token to retrieve
	resource, err = controller.getEntitlementResourceRequestPayload(service.Context, accessTokenString)
	require.Nil(t, err)

	require.NotNil(t, resource)
	require.NotNil(t, resource.Permissions)
	assert.Len(t, resource.Permissions, maxRecentSpacesForRPT)

}

// Load returns a single Identity as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *MockIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*account.Identity, error) {
	return m.testIdentity, nil
}

// Exists returns true|false whether an identity exists with a specific identifier
func (m *MockIdentityRepository) Exists(ctx context.Context, id string) (bool, error) {
	return true, nil
}

// Create creates a new record.
func (m *MockIdentityRepository) Create(ctx context.Context, model *account.Identity) error {
	return nil
}

// Lookup looks for an existing identity with the given `profileURL` or creates a new one
func (m *MockIdentityRepository) Lookup(ctx context.Context, username, profileURL, providerType string) (*account.Identity, error) {
	return m.testIdentity, nil
}

// Save modifies a single record.
func (m *MockIdentityRepository) Save(ctx context.Context, model *account.Identity) error {
	m.testIdentity = model
	return nil
}

// Delete removes a single record.
func (m *MockIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Query expose an open ended Query model
func (m *MockIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.Identity, error) {
	var identities []account.Identity
	identities = append(identities, *m.testIdentity)
	return identities, nil
}

// First returns the first Identity element that matches the given criteria
func (m *MockIdentityRepository) First(funcs ...func(*gorm.DB) *gorm.DB) (*account.Identity, error) {
	return m.testIdentity, nil
}

func (m *MockIdentityRepository) List(ctx context.Context) ([]account.Identity, error) {
	var rows []account.Identity
	rows = append(rows, *m.testIdentity)
	return rows, nil
}

func (m *MockIdentityRepository) CheckExists(ctx context.Context, id string) error {
	return nil
}

func (m *MockIdentityRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	return true
}

func (m *MockIdentityRepository) Search(ctx context.Context, q string, start int, limit int) ([]account.Identity, int, error) {
	result := []account.Identity{}
	result = append(result, *m.testIdentity)
	return result, 1, nil
}

type MockUserRepository struct {
	User *account.User
}

func (m MockUserRepository) Load(ctx context.Context, id uuid.UUID) (*account.User, error) {
	if m.User == nil {
		return nil, errors.New("not found")
	}
	return m.User, nil
}

func (m MockUserRepository) Exists(ctx context.Context, id string) (bool, error) {
	if m.User == nil {
		return false, errors.New("not found")
	}
	return true, nil
}

// Create creates a new record.
func (m MockUserRepository) Create(ctx context.Context, u *account.User) error {
	m.User = u
	return nil
}

// Save modifies a single record
func (m MockUserRepository) Save(ctx context.Context, model *account.User) error {
	return m.Create(ctx, model)
}

// Save modifies a single record
func (m MockUserRepository) CheckExists(ctx context.Context, id string) error {
	return nil
}

// Delete removes a single record.
func (m MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.User = nil
	return nil
}

// List return all users
func (m MockUserRepository) List(ctx context.Context) ([]account.User, error) {
	return []account.User{*m.User}, nil
}

// Query expose an open ended Query model
func (m MockUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.User, error) {
	return []account.User{*m.User}, nil
}
