package controller_test

import (
	"fmt"
	"testing"
	"time"

	"context"

	"net/http"

	token "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestUserREST struct {
	suite.Suite
	config configuration.ConfigurationData
}

func (rest *TestUserREST) TestRunUserREST(t *testing.T) {
	resource.Require(rest.T(), resource.UnitTest)
	t.Parallel()
	suite.Run(rest.T(), &TestUserREST{})
}

func (rest *TestUserREST) SetupSuite() {
	config, err := configuration.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	rest.config = *config
}

func (rest *TestUserREST) newUserController(identity *account.Identity, user *account.User) *UserController {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	return NewUserController(goa.New("wit-test"), newGormTestBase(identity, user), wittoken.NewManagerWithPrivateKey(priv), &rest.config)
}

func (rest *TestUserREST) TestCurrentAuthorizedMissingUUID() {
	resource.Require(rest.T(), resource.UnitTest)
	jwtToken := token.New(token.SigningMethodRS256)
	ctx := jwt.WithJWT(context.Background(), jwtToken)

	userCtrl := rest.newUserController(nil, nil)
	test.ShowUserBadRequest(rest.T(), ctx, nil, userCtrl, nil, nil)
}

func (rest *TestUserREST) TestCurrentAuthorizedNonUUID() {
	// given
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = "aa"
	ctx := jwt.WithJWT(context.Background(), jwtToken)
	// when
	userCtrl := rest.newUserController(nil, nil)
	// then
	test.ShowUserBadRequest(rest.T(), ctx, nil, userCtrl, nil, nil)
}

func (rest *TestUserREST) TestCurrentAuthorizedMissingIdentity() {
	// given
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)
	// when
	userCtrl := rest.newUserController(nil, nil)
	// then
	test.ShowUserUnauthorized(rest.T(), ctx, nil, userCtrl, nil, nil)
}

func (rest *TestUserREST) TestCurrentAuthorizedOK() {
	// given
	ctx, userCtrl, usr, ident := rest.initTestCurrentAuthorized()
	// when
	res, user := test.ShowUserOK(rest.T(), ctx, nil, userCtrl, nil, nil)
	// then
	rest.assertCurrentUser(*user, ident, usr)
	rest.assertResponseHeaders(res, usr)
}

func (rest *TestUserREST) TestCurrentAuthorizedOKUsingExpiredIfModifiedSinceHeader() {
	// given
	ctx, userCtrl, usr, ident := rest.initTestCurrentAuthorized()
	// when
	ifModifiedSince := usr.UpdatedAt.Add(-1 * time.Hour).UTC().Format(http.TimeFormat)
	res, user := test.ShowUserOK(rest.T(), ctx, nil, userCtrl, &ifModifiedSince, nil)
	// then
	rest.assertCurrentUser(*user, ident, usr)
	rest.assertResponseHeaders(res, usr)
}

func (rest *TestUserREST) TestCurrentAuthorizedOKUsingExpiredIfNoneMatchHeader() {
	// given
	ctx, userCtrl, usr, ident := rest.initTestCurrentAuthorized()
	// when
	ifNoneMatch := "foo"
	res, user := test.ShowUserOK(rest.T(), ctx, nil, userCtrl, nil, &ifNoneMatch)
	// then
	rest.assertCurrentUser(*user, ident, usr)
	rest.assertResponseHeaders(res, usr)
}

func (rest *TestUserREST) TestCurrentAuthorizedNotModifiedUsingIfModifiedSinceHeader() {
	// given
	ctx, userCtrl, usr, _ := rest.initTestCurrentAuthorized()
	// when
	ifModifiedSince := usr.UpdatedAt.Add(-1 * time.Hour).UTC().Format(http.TimeFormat)
	res := test.ShowUserNotModified(rest.T(), ctx, nil, userCtrl, &ifModifiedSince, nil)
	// then
	rest.assertResponseHeaders(res, usr)
}

func (rest *TestUserREST) TestCurrentAuthorizedNotModifiedUsingIfNoneMatchHeader() {
	// given
	ctx, userCtrl, usr, _ := rest.initTestCurrentAuthorized()
	// when
	ifNoneMatch := "foo"
	res := test.ShowUserNotModified(rest.T(), ctx, nil, userCtrl, nil, &ifNoneMatch)
	// then
	rest.assertResponseHeaders(res, usr)
}

func (rest *TestUserREST) initTestCurrentAuthorized() (context.Context, app.UserController, account.User, account.Identity) {
	jwtToken := token.New(token.SigningMethodRS256)
	jwtToken.Claims.(token.MapClaims)["sub"] = uuid.NewV4().String()
	ctx := jwt.WithJWT(context.Background(), jwtToken)
	usr := account.User{
		ID: uuid.NewV4(),
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		FullName: "TestCurrentAuthorizedOK User",
		ImageURL: "someURL",
		Email:    "email@domain.com",
	}
	ident := account.Identity{ID: uuid.NewV4(), Username: "TestUser", ProviderType: account.KeycloakIDP, User: usr, UserID: account.NullUUID{UUID: usr.ID, Valid: true}}
	userCtrl := rest.newUserController(&ident, &usr)
	return ctx, userCtrl, usr, ident
}

func (rest *TestUserREST) assertCurrentUser(user app.User, ident account.Identity, usr account.User) {
	require.NotNil(rest.T(), user)
	require.NotNil(rest.T(), user.Data)
	require.NotNil(rest.T(), user.Data.Attributes)
	assert.Equal(rest.T(), usr.FullName, *user.Data.Attributes.FullName)
	assert.Equal(rest.T(), ident.Username, *user.Data.Attributes.Username)
	assert.Equal(rest.T(), usr.ImageURL, *user.Data.Attributes.ImageURL)
	assert.Equal(rest.T(), usr.Email, *user.Data.Attributes.Email)
	assert.Equal(rest.T(), ident.ProviderType, *user.Data.Attributes.ProviderType)
}

func (rest *TestUserREST) assertResponseHeaders(res http.ResponseWriter, usr account.User) {
	require.NotNil(rest.T(), res.Header()[app.LastModified])
	assert.Equal(rest.T(), usr.UpdatedAt.Truncate(time.Second).UTC().Format(http.TimeFormat), res.Header()[app.LastModified][0])
	require.NotNil(rest.T(), res.Header()[app.CacheControl])
	assert.Equal(rest.T(), rest.config.GetCacheControlUser(), res.Header()[app.CacheControl][0])
	require.NotNil(rest.T(), res.Header()[app.ETag])
	assert.Equal(rest.T(), app.GenerateEntityTag(usr), res.Header()[app.ETag][0])

}

type TestIdentityRepository struct {
	Identity *account.Identity
}

// Load returns a single Identity as a Database Model
func (m TestIdentityRepository) Load(ctx context.Context, id uuid.UUID) (*account.Identity, error) {
	if m.Identity == nil {
		return nil, errors.New("not found")
	}
	return m.Identity, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (m TestIdentityRepository) CheckExists(ctx context.Context, id string) error {
	if m.Identity == nil {
		return errors.New("not found")
	}
	return nil
}

// Create creates a new record.
func (m TestIdentityRepository) Create(ctx context.Context, model *account.Identity) error {
	m.Identity = model
	return nil
}

// Lookup looks up a record or creates a new one.
func (m TestIdentityRepository) Lookup(ctx context.Context, username, profileURL, providerType string) (*account.Identity, error) {
	return nil, nil
}

// Lookup looks up a record or creates a new one.
func (m TestIdentityRepository) Search(ctx context.Context, q string, start int, limit int) ([]account.Identity, int, error) {
	return nil, 0, nil
}

// Save modifies a single record.
func (m TestIdentityRepository) Save(ctx context.Context, model *account.Identity) error {
	return m.Create(ctx, model)
}

// Delete removes a single record.
func (m TestIdentityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.Identity = nil
	return nil
}

// Query expose an open ended Query model
func (m TestIdentityRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.Identity, error) {
	return []account.Identity{*m.Identity}, nil
}

func (m TestIdentityRepository) List(ctx context.Context) ([]account.Identity, error) {
	rows := []account.Identity{*m.Identity}
	return rows, nil
}

func (m TestIdentityRepository) IsValid(ctx context.Context, id uuid.UUID) bool {
	return true
}

type TestUserRepository struct {
	User *account.User
}

func (m TestUserRepository) Load(ctx context.Context, id uuid.UUID) (*account.User, error) {
	if m.User == nil {
		return nil, errors.New("not found")
	}
	return m.User, nil
}

func (m TestUserRepository) CheckExists(ctx context.Context, id string) error {
	if m.User == nil {
		return errors.New("not found")
	}
	return nil
}

// Create creates a new record.
func (m TestUserRepository) Create(ctx context.Context, u *account.User) error {
	m.User = u
	return nil
}

// Save modifies a single record
func (m TestUserRepository) Save(ctx context.Context, model *account.User) error {
	return m.Create(ctx, model)
}

// Delete removes a single record.
func (m TestUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	m.User = nil
	return nil
}

// List return all users
func (m TestUserRepository) List(ctx context.Context) ([]account.User, error) {
	return []account.User{*m.User}, nil
}

// Query expose an open ended Query model
func (m TestUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]account.User, error) {
	return []account.User{*m.User}, nil
}

type GormTestBase struct {
	IdentityRepository account.IdentityRepository
	UserRepository     account.UserRepository
}

func (g *GormTestBase) WorkItems() workitem.WorkItemRepository {
	return nil
}

func (g *GormTestBase) WorkItemTypes() workitem.WorkItemTypeRepository {
	return nil
}

func (g *GormTestBase) Spaces() space.Repository {
	return nil
}

func (g *GormTestBase) SpaceResources() space.ResourceRepository {
	return nil
}

func (g *GormTestBase) Trackers() application.TrackerRepository {
	return nil
}
func (g *GormTestBase) TrackerQueries() application.TrackerQueryRepository {
	return nil
}

func (g *GormTestBase) SearchItems() application.SearchRepository {
	return nil
}

// Identities creates new Identity repository
func (g *GormTestBase) Identities() account.IdentityRepository {
	return g.IdentityRepository
}

// Users creates new user repository
func (g *GormTestBase) Users() account.UserRepository {
	return g.UserRepository
}

// WorkItemLinkCategories returns a work item link category repository
func (g *GormTestBase) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return nil
}

// WorkItemLinkTypes returns a work item link type repository
func (g *GormTestBase) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return nil
}

// WorkItemLinks returns a work item link repository
func (g *GormTestBase) WorkItemLinks() link.WorkItemLinkRepository {
	return nil
}

// Comments returns a work item comments repository
func (g *GormTestBase) Comments() comment.Repository {
	return nil
}

// Iterations returns a iteration repository
func (g *GormTestBase) Iterations() iteration.Repository {
	return nil
}

// Iterations returns a iteration repository
func (g *GormTestBase) Areas() area.Repository {
	return nil
}

func (g *GormTestBase) OauthStates() auth.OauthStateReferenceRepository {
	return nil
}

// Codebases returns a codebase repository
func (g *GormTestBase) Codebases() codebase.Repository {
	return nil
}

func (g *GormTestBase) DB() *gorm.DB {
	return nil
}

// SetTransactionIsolationLevel sets the isolation level for
// See also https://www.postgresql.org/docs/9.3/static/sql-set-transaction.html
func (g *GormTestBase) SetTransactionIsolationLevel(level interface{}) error {
	return nil
}

func (g *GormTestBase) Commit() error {
	return nil
}

func (g *GormTestBase) Rollback() error {
	return nil
}

// Begin implements TransactionSupport
func (g *GormTestBase) BeginTransaction() (application.Transaction, error) {
	return g, nil
}

func newGormTestBase(identity *account.Identity, user *account.User) *GormTestBase {
	return &GormTestBase{
		IdentityRepository: TestIdentityRepository{Identity: identity},
		UserRepository:     TestUserRepository{User: user}}
}
