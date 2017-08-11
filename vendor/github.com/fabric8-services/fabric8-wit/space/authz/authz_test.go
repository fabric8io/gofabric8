package authz_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/goadesign/goa"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	scopes = []string{"read:test", "admin:test"}
)

func TestAuthz(t *testing.T) {
	resource.Require(t, resource.Remote)
	suite.Run(t, new(TestAuthzSuite))
}

type TestAuthzSuite struct {
	suite.Suite
	configuration *config.ConfigurationData
	authzService  *authz.KeycloakAuthzService
}

func (s *TestAuthzSuite) SetupSuite() {
	var err error
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
	var resource *space.Resource
	s.configuration, err = config.GetConfigurationData()
	s.authzService = authz.NewAuthzService(s.configuration, &db{app{resource: resource}})
}

func (s *TestAuthzSuite) TestFailsIfNoTokenInContext() {
	ctx := context.Background()
	spaceID := ""
	_, err := s.authzService.Authorize(ctx, "", spaceID)
	require.NotNil(s.T(), err)
}

func (s *TestAuthzSuite) TestUserAmongSpaceCollaboratorsOK() {
	spaceID := uuid.NewV4().String()
	authzPayload := auth.AuthorizationPayload{Permissions: []auth.Permissions{{ResourceSetName: &spaceID}}}
	ok := s.checkPermissions(authzPayload, spaceID)
	require.True(s.T(), ok)
}

func (s *TestAuthzSuite) TestUserIsNotAmongSpaceCollaboratorsFails() {
	spaceID1 := uuid.NewV4().String()
	spaceID2 := uuid.NewV4().String()
	authzPayload := auth.AuthorizationPayload{Permissions: []auth.Permissions{{ResourceSetName: &spaceID1}}}
	ok := s.checkPermissions(authzPayload, spaceID2)
	require.False(s.T(), ok)
}

func (s *TestAuthzSuite) checkPermissions(authzPayload auth.AuthorizationPayload, spaceID string) bool {
	resource := &space.Resource{}
	authzService := authz.NewAuthzService(nil, &db{app{resource: resource}})
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	testIdentity := testsupport.TestIdentity
	svc := testsupport.ServiceAsUserWithAuthz("SpaceAuthz-Service", wittoken.NewManagerWithPrivateKey(priv), priv, testIdentity, authzPayload)
	resource.UpdatedAt = time.Now()
	r := &goa.RequestData{
		Request: &http.Request{Host: "api.example.org"},
	}
	entitlementEndpoint, err := s.configuration.GetKeycloakEndpointEntitlement(r)
	require.Nil(s.T(), err)
	ok, err := authzService.Authorize(svc.Context, entitlementEndpoint, spaceID)
	require.Nil(s.T(), err)
	return ok
}

type app struct {
	resource *space.Resource
}

type db struct {
	app
}

type trx struct {
	app
}

type resourceRepo struct {
	resource *space.Resource
}

func (t *trx) Commit() error {
	return nil
}

func (t *trx) Rollback() error {
	return nil
}

func (d *db) BeginTransaction() (application.Transaction, error) {
	return &trx{}, nil
}

func (a *app) WorkItems() workitem.WorkItemRepository {
	return nil
}

func (a *app) WorkItemTypes() workitem.WorkItemTypeRepository {
	return nil
}

func (a *app) Trackers() application.TrackerRepository {
	return nil
}

func (a *app) TrackerQueries() application.TrackerQueryRepository {
	return nil
}

func (a *app) SearchItems() application.SearchRepository {
	return nil
}

func (a *app) Identities() account.IdentityRepository {
	return nil
}

func (a *app) WorkItemLinkCategories() link.WorkItemLinkCategoryRepository {
	return nil
}

func (a *app) WorkItemLinkTypes() link.WorkItemLinkTypeRepository {
	return nil
}

func (a *app) WorkItemLinks() link.WorkItemLinkRepository {
	return nil
}

func (a *app) Comments() comment.Repository {
	return nil
}

func (a *app) Spaces() space.Repository {
	return nil
}

func (a *app) SpaceResources() space.ResourceRepository {
	return &resourceRepo{a.resource}
}

func (a *app) Iterations() iteration.Repository {
	return nil
}

func (a *app) Users() account.UserRepository {
	return nil
}

func (a *app) Areas() area.Repository {
	return nil
}

func (a *app) OauthStates() auth.OauthStateReferenceRepository {
	return nil
}

func (a *app) Codebases() codebase.Repository {
	return nil
}

func (r *resourceRepo) Create(ctx context.Context, s *space.Resource) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Save(ctx context.Context, s *space.Resource) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Load(ctx context.Context, ID uuid.UUID) (*space.Resource, error) {
	return nil, nil
}

func (r *resourceRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	return nil
}

func (r *resourceRepo) CheckExists(ctx context.Context, ID string) error {
	return nil
}

func (r *resourceRepo) LoadBySpace(ctx context.Context, spaceID *uuid.UUID) (*space.Resource, error) {
	resource := &space.Resource{}
	past := time.Now().Unix() - 1000
	resource.UpdatedAt = time.Unix(past, 0)
	return resource, nil
}
