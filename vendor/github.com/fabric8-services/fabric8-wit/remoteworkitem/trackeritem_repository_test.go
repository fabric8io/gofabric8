package remoteworkitem

import (
	"net/http"
	"net/url"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteTrackerItemRepository
func TestSuiteTrackerItemRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TrackerItemRepositorySuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TrackeItemRepositorySuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type TrackerItemRepositorySuite struct {
	gormtestsupport.DBTestSuite
	clean        func()
	trackerQuery TrackerQuery
	ctx          context.Context
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *TrackerItemRepositorySuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *TrackerItemRepositorySuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	// Setting up the dependent tracker query and tracker data in the Database
	tracker := Tracker{URL: "https://api.github.com/", Type: ProviderGithub}
	s.trackerQuery = TrackerQuery{Query: "some random query", Schedule: "0 0 0 * * *", TrackerID: tracker.ID, SpaceID: space.SystemSpace}

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	s.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (s *TrackerItemRepositorySuite) createIdentity(username string) account.Identity {
	identityRepo := account.NewIdentityRepository(s.DB)
	profile := "https://api.github.com/users/" + username
	identity := account.Identity{
		Username:     username,
		ProfileURL:   &profile,
		ProviderType: ProviderGithub,
	}
	err := identityRepo.Create(s.ctx, &identity)
	require.Nil(s.T(), err)
	return identity
}

func (s *TrackerItemRepositorySuite) lookupIdentityByID(id string) account.Identity {
	identityRepo := account.NewIdentityRepository(s.DB)
	identityID, err := uuid.FromString(id)
	require.Nil(s.T(), err)
	identity, err := identityRepo.First(account.IdentityFilterByID(identityID))
	require.Nil(s.T(), err)
	return *identity
}

func (s *TrackerItemRepositorySuite) TearDownTest() {
	s.clean()
}

var GitIssueWithAssignee = "http://api.github.com/repos/fabric8-wit-test/fabric8-wit-test-unit/issues/2"

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItemWithExistingIdentities() {
	// given
	identity0 := s.createIdentity("jdoe0")
	identity1 := s.createIdentity("jdoe1")
	identity2 := s.createIdentity("jdoe2")
	remoteItemData := TrackerItemContent{
		Content: []byte(`
				{
					"title": "linking",
					"url": "http://github.com/sbose/api/testonly/1",
					"state": "closed",
					"body": "body of issue",
					"user": {
						"login": "jdoe0",
						"url": "https://api.github.com/users/jdoe0"
					},
					"assignees": [
						{
							"login": "jdoe1",
							"url": "https://api.github.com/users/jdoe1"
						},
						{
							"login": "jdoe2",
							"url": "https://api.github.com/users/jdoe2"
						}]
				}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}

	// when
	workItem, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	require.Nil(s.T(), err)
	require.NotNil(s.T(), workItem.Fields)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), identity0.ID.String(), workItem.Fields[workitem.SystemCreator])
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Contains(s.T(), workItem.Fields[workitem.SystemAssignees], identity1.ID.String())
	assert.Contains(s.T(), workItem.Fields[workitem.SystemAssignees], identity2.ID.String())
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
}

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItemWithUnknownIdentities() {
	// given "jdoe" identity does not exist
	remoteItemData := TrackerItemContent{
		Content: []byte(`
				{
					"title": "linking",
					"url": "http://github.com/sbose/api/testonly/1",
					"state": "closed",
					"body": "body of issue",
					"user": {
						"login": "jdoe0",
						"url": "https://api.github.com/users/jdoe0"
					},
					"assignees": [
						{
							"login": "jdoe1",
							"url": "https://api.github.com/users/jdoe1"
						},
						{
							"login": "jdoe2",
							"url": "https://api.github.com/users/jdoe2"
						}]
				}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}

	// when
	workItem, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	require.Nil(s.T(), err)
	require.NotNil(s.T(), workItem.Fields)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), 2, len(workItem.Fields[workitem.SystemAssignees].([]interface{})))
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
	// look-up creator identity in repository
	require.NotNil(s.T(), workItem.Fields[workitem.SystemCreator])
	identityID := workItem.Fields[workitem.SystemCreator]
	assert.NotNil(s.T(), s.lookupIdentityByID(identityID.(string)))
	// look-up assignee identities in repository
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	identityIDs := workItem.Fields[workitem.SystemAssignees].([]interface{})
	for _, identityID := range identityIDs {
		identity := s.lookupIdentityByID(identityID.(string))
		require.NotNil(s.T(), identity)
		assert.Contains(s.T(), identity.Username, "jdoe")
		assert.NotContains(s.T(), identity.Username, "https://api.github.com/users/jdoe")
		assert.Contains(s.T(), *identity.ProfileURL, "https://api.github.com/users/jdoe")
	}
}

func (s *TrackerItemRepositorySuite) TestConvertNewWorkItemWithNoAssignee() {
	// given
	identity0 := s.createIdentity("jdoe0")
	remoteItemData := TrackerItemContent{
		Content: []byte(`
				{
					"title": "linking",
					"url": "http://github.com/sbose/api/testonly/1",
					"state": "closed",
					"body": "body of issue",
					"user": {
						"login": "jdoe0",
						"url": "https://api.github.com/users/jdoe0"
					},
					"assignees": []
				}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItem, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	require.Nil(s.T(), err)
	require.NotNil(s.T(), workItem.Fields)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Empty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	require.NotNil(s.T(), workItem.Fields[workitem.SystemDescription])
	description := workItem.Fields[workitem.SystemDescription].(rendering.MarkupContent)
	assert.Equal(s.T(), "body of issue", description.Content)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, description.Markup)
	// look-up creator identity in repository
	require.NotNil(s.T(), workItem.Fields[workitem.SystemCreator])
	identityID := workItem.Fields[workitem.SystemCreator]
	assert.Equal(s.T(), identity0.ID, s.lookupIdentityByID(identityID.(string)).ID)
}

func (s *TrackerItemRepositorySuite) TestConvertExistingWorkItem() {
	// given
	identity0 := s.createIdentity("jdoe0")
	identity1 := s.createIdentity("jdoe1")
	identity2 := s.createIdentity("jdoe2")
	remoteItemData := TrackerItemContent{
		// content is already flattened
		Content: []byte(`
			{
				"title": "linking",
				"url": "http://github.com/sbose/api/testonly/1",
				"state": "closed",
				"body": "body of issue",
				"user.login": "jdoe0",
				"user.url": "https://api.github.com/users/jdoe0",
				"assignees.0.login": "jdoe1",
				"assignees.0.url": "https://api.github.com/users/jdoe1",
				"assignees.1.login": "jdoe2",
				"assignees.1.url": "https://api.github.com/users/jdoe2"
			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItem, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemData, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "linking", workItem.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), identity0.ID.String(), workItem.Fields[workitem.SystemCreator])
	require.NotEmpty(s.T(), workItem.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), identity1.ID.String(), workItem.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), identity2.ID.String(), workItem.Fields[workitem.SystemAssignees].([]interface{})[1])
	assert.Equal(s.T(), "closed", workItem.Fields[workitem.SystemState])
	// given
	s.T().Log("Updating the existing work item when it's reimported.")
	identity3 := s.createIdentity("jdoe3")
	identity4 := s.createIdentity("jdoe4")
	remoteItemDataUpdated := TrackerItemContent{
		// content is already flattened
		Content: []byte(`
			{
				"title": "linking-updated",
				"url": "http://github.com/sbose/api/testonly/1",
				"state": "closed",
				"body": "body of issue",
				"user.login": "jdoe3",
				"user.url": "https://api.github.com/users/jdoe3",
				"assignees.0.login": "jdoe4",
				"assignees.0.url": "https://api.github.com/users/jdoe4"
			}`),
		ID: "http://github.com/sbose/api/testonly/1",
	}
	// when
	workItemUpdated, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemDataUpdated, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), workItemUpdated)
	require.NotNil(s.T(), workItemUpdated.Fields)
	assert.Equal(s.T(), "linking-updated", workItemUpdated.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), identity3.ID.String(), workItemUpdated.Fields[workitem.SystemCreator])
	require.NotEmpty(s.T(), workItemUpdated.Fields[workitem.SystemAssignees])
	assert.Equal(s.T(), identity4.ID.String(), workItemUpdated.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "closed", workItemUpdated.Fields[workitem.SystemState])
}

func (s *TrackerItemRepositorySuite) TestConvertGithubIssue() {
	// given
	identity := s.createIdentity("sbose78")
	content, err := test.LoadTestData("github_issue_mapping.json", func() ([]byte, error) {
		return provideRemoteData(GitIssueWithAssignee)
	})
	require.Nil(s.T(), err)
	remoteItemDataGithub := TrackerItemContent{
		Content: content[:],
		ID:      GitIssueWithAssignee, // GH issue url
	}
	// when
	workItemGithub, err := convertToWorkItemModel(s.ctx, s.DB, int(s.trackerQuery.ID), remoteItemDataGithub, ProviderGithub, s.trackerQuery.SpaceID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), "map flatten : test case : with assignee", workItemGithub.Fields[workitem.SystemTitle])
	assert.Equal(s.T(), identity.ID.String(), workItemGithub.Fields[workitem.SystemCreator])
	assert.Equal(s.T(), identity.ID.String(), workItemGithub.Fields[workitem.SystemAssignees].([]interface{})[0])
	assert.Equal(s.T(), "open", workItemGithub.Fields[workitem.SystemState])
}
