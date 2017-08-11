package controller_test

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/comment"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteComments
func TestSuiteComments(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &CommentsSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TestSuiteComments struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CommentsSuite struct {
	gormtestsupport.DBTestSuite
	db            *gormapplication.GormDB
	clean         func()
	testIdentity  account.Identity
	testIdentity2 account.Identity
	notification  testsupport.NotificationChannel
}

func (s *CommentsSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *CommentsSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user2", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity2 = *testIdentity2
	s.notification = testsupport.NotificationChannel{}
}

func (s *CommentsSuite) TearDownTest() {
	s.clean()
}

var (
	markdownMarkup  = rendering.SystemMarkupMarkdown
	plaintextMarkup = rendering.SystemMarkupPlainText
	defaultMarkup   = rendering.SystemMarkupDefault
)

func (s *CommentsSuite) unsecuredController() (*goa.Service, *CommentsController) {
	svc := goa.New("Comments-service-test")
	commentsCtrl := NewNotifyingCommentsController(svc, s.db, &s.notification, s.Configuration)
	return svc, commentsCtrl
}

func (s *CommentsSuite) securedControllers(identity account.Identity) (*goa.Service, *WorkitemController, *WorkitemsController, *WorkItemCommentsController, *CommentsController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Comment-Service", wittoken.NewManagerWithPrivateKey(priv), identity)
	workitemCtrl := NewWorkitemController(svc, s.db, s.Configuration)
	workitemsCtrl := NewWorkitemsController(svc, s.db, s.Configuration)
	workitemCommentsCtrl := NewWorkItemCommentsController(svc, s.db, s.Configuration)

	commentsCtrl := NewNotifyingCommentsController(svc, s.db, &s.notification, s.Configuration)
	return svc, workitemCtrl, workitemsCtrl, workitemCommentsCtrl, commentsCtrl
}

// createWorkItem creates a workitem that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItem(identity account.Identity) uuid.UUID {
	spaceRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	witRelatedURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(space.SystemSpace.String(), workitem.SystemBug.String()))
	createWorkitemPayload := app.CreateWorkitemsPayload{
		Data: &app.WorkItem{
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "work item title",
				workitem.SystemState: workitem.SystemStateNew},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   workitem.SystemBug,
					},
					Links: &app.GenericLinks{
						Self:    &witRelatedURL,
						Related: &witRelatedURL,
					},
				},
				Space: app.NewSpaceRelation(space.SystemSpace, spaceRelatedURL),
			},
		},
	}
	userSvc, _, workitemsCtrl, _, _ := s.securedControllers(identity)
	_, wi := test.CreateWorkitemsCreated(s.T(), userSvc.Context, userSvc, workitemsCtrl, *createWorkitemPayload.Data.Relationships.Space.Data.ID, &createWorkitemPayload)
	wiID := *wi.Data.ID
	s.T().Log(fmt.Sprintf("Created workitem with id %v", wiID))
	return wiID
}

func newCreateWorkItemCommentsPayload(body string, markup *string) *app.CreateWorkItemCommentsPayload {
	return &app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body:   body,
				Markup: markup,
			},
		},
	}
}

// createWorkItemComment creates a workitem comment that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItemComment(identity account.Identity, wiID uuid.UUID, body string, markup *string) app.CommentSingle {
	createWorkItemCommentPayload := newCreateWorkItemCommentsPayload(body, markup)
	userSvc, _, _, workitemCommentsCtrl, _ := s.securedControllers(identity)
	_, c := test.CreateWorkItemCommentsOK(s.T(), userSvc.Context, userSvc, workitemCommentsCtrl, wiID, createWorkItemCommentPayload)
	require.NotNil(s.T(), c)
	s.T().Log(fmt.Sprintf("Created comment with id %v", *c.Data.ID))
	return *c
}

func newUpdateCommentsPayload(body string, markup *string) *app.UpdateCommentsPayload {
	return &app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body:   &body,
				Markup: markup,
			},
		},
	}
}

// updateComment updates the comment with the given commentId
func (s *CommentsSuite) updateComment(identity account.Identity, commentID uuid.UUID, body string, markup *string) app.CommentSingle {
	updateCommentsPayload := newUpdateCommentsPayload(body, markup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(identity)
	_, c := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentID, updateCommentsPayload)
	require.NotNil(s.T(), c)
	s.T().Log(fmt.Sprintf("Updated comment with id %v", *c.Data.ID))
	return *c
}

// deleteComment deletes the comment with the given commentId
func (s *CommentsSuite) deleteComment(identity account.Identity, commentID uuid.UUID) {
	userSvc, _, _, _, commentsCtrl := s.securedControllers(identity)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, commentID)
	s.T().Log(fmt.Sprintf("Deleted comment with id %v", commentID))
}

func assertComment(t *testing.T, resultData *app.Comment, expectedIdentity account.Identity, expectedBody string, expectedMarkup string) {
	require.NotNil(t, resultData)
	assert.NotNil(t, resultData.ID)
	assert.NotNil(t, resultData.Type)
	require.NotNil(t, resultData.Attributes)
	require.NotNil(t, resultData.Attributes.CreatedAt)
	require.NotNil(t, resultData.Attributes.UpdatedAt)
	require.NotNil(t, resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Body)
	assert.Equal(t, expectedBody, *resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Markup)
	assert.Equal(t, expectedMarkup, *resultData.Attributes.Markup)
	assert.Equal(t, rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *resultData.Attributes.BodyRendered)
	require.NotNil(t, resultData.Relationships)
	require.NotNil(t, resultData.Relationships.Creator)
	require.NotNil(t, resultData.Relationships.Creator.Data)
	require.NotNil(t, resultData.Relationships.Creator.Data.ID)
	assert.Equal(t, expectedIdentity.ID, uuid.FromStringOrNil(*resultData.Relationships.Creator.Data.ID))
	assert.True(t, strings.Contains(*resultData.Relationships.Creator.Data.ID, *resultData.Relationships.Creator.Data.ID), "Link not found")
}

func convertCommentToModel(c app.CommentSingle) comment.Comment {
	return comment.Comment{
		ID: *c.Data.ID,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *c.Data.Attributes.UpdatedAt,
		},
	}
}

func (s *CommentsSuite) TestShowCommentWithBackwardSupport() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assert.Equal(s.T(), *result.Data.Relationships.Creator.Data.ID, result.Data.Relationships.CreatedBy.Data.ID.String())
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOK() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfModifiedSinceHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifModifiedSince := app.ToHTTPTime(c.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfNoneMatchHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifNoneMatch := "foo"
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfModifiedSinceHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifModifiedSince := app.ToHTTPTime(*c.Data.Attributes.UpdatedAt)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfNoneMatchHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	commentModel := convertCommentToModel(c)
	ifNoneMatch := app.GenerateEntityTag(commentModel)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthWithMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithEscapedScriptInjection() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "<img src=x onerror=alert('body') />", &plaintextMarkup)
	// when
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "<img src=x onerror=alert('body') />", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithOtherMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithNilMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", nil)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupDefault)
}

func (s *CommentsSuite) TestDeleteCommentWithSameAuthenticatedUser() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	// when/then
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestDeleteCommentWithoutAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	// when/then
	test.DeleteCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

// Following test creates a space and space_owner creates a WI in that space
// Space owner adds a comment on the created WI
// Create another user, which is not space collaborator.
// Test if another user can delete the comment
func (s *CommentsSuite) TestNonCollaboraterCanNotDelete() {
	// create space
	// create user
	// add user to the space collaborator list
	// create workitem in created space
	// create another user - do not add this user into collaborator list
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotDelete-"), "TestWIComments")
	require.Nil(s.T(), err)
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *testIdentity)

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity, &TestSpaceAuthzService{*testIdentity})
	workitemsCtrl := NewWorkitemsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup)

	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotDelete-"), "TestWI")
	svcNotAuthorized := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity2, &TestSpaceAuthzService{*testIdentity})
	commentsCtrlNotAuthorized := NewCommentsController(svcNotAuthorized, gormapplication.NewGormDB(s.DB), s.Configuration)

	test.DeleteCommentsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, commentsCtrlNotAuthorized, *c.Data.ID)
}

func (s *CommentsSuite) TestCollaboratorCanDelete() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestCollaboratorCanDelete-"), "TestWIComments")
	require.Nil(s.T(), err)
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *testIdentity)

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	svc, _, workitemsCtrl, _, _ := s.securedControllers(*testIdentity)
	// priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	// svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), testIdentity, &TestSpaceAuthzService{testIdentity})
	// ctrl := NewWorkitemsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup)
	commentCtrl := NewCommentsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	test.DeleteCommentsOK(s.T(), svc.Context, svc, commentCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestCreatorCanDelete() {
	wID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wID, "body", &plaintextMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestOtherCollaboratorCanDelete() {
	// create space owner identity
	spaceOwner, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.Nil(s.T(), err)

	// create 2 space collaborators' identity
	collaborator1, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.Nil(s.T(), err)

	collaborator2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanDelete-"), "TestWIComments")
	require.Nil(s.T(), err)

	// Add 2 identities as Collaborators
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *spaceOwner)
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svcWithSpaceOwner := testsupport.ServiceAsSpaceUser("Comments-Service", wittoken.NewManagerWithPrivateKey(priv), *spaceOwner, &TestSpaceAuthzService{*spaceOwner})
	collaboratorRESTInstance := &TestCollaboratorsREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")}
	collaboratorRESTInstance.policy = &auth.KeycloakPolicy{
		Name:             "TestCollaborators-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	collaboratorCtrl := NewCollaboratorsController(svcWithSpaceOwner, s.db, s.Configuration, &DummyPolicyManager{rest: collaboratorRESTInstance})
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *space.ID, collaborator1.ID.String())
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *space.ID, collaborator2.ID.String())

	// Build WI payload and create 1 WI (created_by = space owner)
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	workitemsCtrl := NewWorkitemsController(svcWithSpaceOwner, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// collaborator1 adds a comment on newly created work item
	c := s.createWorkItemComment(*collaborator1, *wi.Data.ID, "Hello woody", &plaintextMarkup)

	// Collaborator2 deletes the comment
	svcWithCollaborator2 := testsupport.ServiceAsSpaceUser("Comments-Service", wittoken.NewManagerWithPrivateKey(priv), *collaborator2, &TestSpaceAuthzService{*collaborator2})
	commentCtrl := NewCommentsController(svcWithCollaborator2, gormapplication.NewGormDB(s.DB), s.Configuration)
	test.DeleteCommentsOK(s.T(), svcWithCollaborator2.Context, svcWithCollaborator2, commentCtrl, *c.Data.ID)
}

// Following test creates a space and space_owner creates a WI in that space
// Space owner adds a comment on the created WI
// Create another user, which is not space collaborator.
// Test if another user can edit/update the comment
func (s *CommentsSuite) TestNonCollaboraterCanNotUpdate() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotUpdate-"), "TestWIComments")
	require.Nil(s.T(), err)
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *testIdentity)

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI 2"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity, &TestSpaceAuthzService{*testIdentity})
	workitemsCtrl := NewWorkitemsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup)

	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestNonCollaboraterCanNotUpdate-"), "TestWI")
	svcNotAuthorized := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), *testIdentity2, &TestSpaceAuthzService{*testIdentity})
	commentCtrlNotAuthorized := NewCommentsController(svcNotAuthorized, gormapplication.NewGormDB(s.DB), s.Configuration)

	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	test.UpdateCommentsForbidden(s.T(), svcNotAuthorized.Context, svcNotAuthorized, commentCtrlNotAuthorized, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestCollaboratorCanUpdate() {
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestCollaboratorCanUpdate-"), "TestWIComments")
	require.Nil(s.T(), err)
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *testIdentity)

	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew

	// priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	// svc := testsupport.ServiceAsSpaceUser("Collaborators-Service", wittoken.NewManagerWithPrivateKey(priv), testIdentity, &TestSpaceAuthzService{testIdentity})
	// ctrl := NewWorkitemController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)
	svc, _, workitemsCtrl, _, _ := s.securedControllers(*testIdentity)

	_, wi := test.CreateWorkitemsCreated(s.T(), svc.Context, svc, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)
	c := s.createWorkItemComment(*testIdentity, *wi.Data.ID, "body", &plaintextMarkup)
	commentCtrl := NewCommentsController(svc, gormapplication.NewGormDB(s.DB), s.Configuration)

	updatedBody := "I am updated comment"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	_, result := test.UpdateCommentsOK(s.T(), svc.Context, svc, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *testIdentity, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestCreatorCanUpdate() {
	wID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wID, "Hello world", &plaintextMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)

	updatedBody := "Hello world in golang"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestOtherCollaboratorCanUpdate() {
	// create space owner identity
	spaceOwner, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.Nil(s.T(), err)

	// create 2 space collaborators' identity
	collaborator1, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.Nil(s.T(), err)

	collaborator2, err := testsupport.CreateTestIdentity(s.DB, testsupport.CreateRandomValidTestName("TestOtherCollaboratorCanUpdate-"), "TestWIComments")
	require.Nil(s.T(), err)

	// Add 2 Collaborators in space
	space := CreateSecuredSpace(s.T(), gormapplication.NewGormDB(s.DB), s.Configuration, *spaceOwner)
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svcWithSpaceOwner := testsupport.ServiceAsSpaceUser("Comments-Service", wittoken.NewManagerWithPrivateKey(priv), *spaceOwner, &TestSpaceAuthzService{*spaceOwner})

	collaboratorRESTInstance := &TestCollaboratorsREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")}
	collaboratorRESTInstance.policy = &auth.KeycloakPolicy{
		Name:             "TestCollaborators-" + uuid.NewV4().String(),
		Type:             auth.PolicyTypeUser,
		Logic:            auth.PolicyLogicPossitive,
		DecisionStrategy: auth.PolicyDecisionStrategyUnanimous,
	}
	collaboratorCtrl := NewCollaboratorsController(svcWithSpaceOwner, s.db, s.Configuration, &DummyPolicyManager{rest: collaboratorRESTInstance})
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *space.ID, collaborator1.ID.String())
	test.AddCollaboratorsOK(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, collaboratorCtrl, *space.ID, collaborator2.ID.String())

	// Build WI payload and create 1 WI (created_by = space owner)
	payload := minimumRequiredCreateWithTypeAndSpace(workitem.SystemFeature, *space.ID)
	payload.Data.Attributes[workitem.SystemTitle] = "Test WI"
	payload.Data.Attributes[workitem.SystemState] = workitem.SystemStateNew
	workitemsCtrl := NewWorkitemsController(svcWithSpaceOwner, gormapplication.NewGormDB(s.DB), s.Configuration)

	_, wi := test.CreateWorkitemsCreated(s.T(), svcWithSpaceOwner.Context, svcWithSpaceOwner, workitemsCtrl, *payload.Data.Relationships.Space.Data.ID, &payload)

	// collaborator1 adds a comment on newly created work item
	c := s.createWorkItemComment(*collaborator1, *wi.Data.ID, "Hello woody", &plaintextMarkup)

	// update comment by collaborator 1
	updatedBody := "Another update on same comment"
	updateCommentPayload := newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	svcWithCollaborator1 := testsupport.ServiceAsSpaceUser("Comments-Service", wittoken.NewManagerWithPrivateKey(priv), *collaborator1, &TestSpaceAuthzService{*collaborator1})
	commentCtrl := NewCommentsController(svcWithCollaborator1, gormapplication.NewGormDB(s.DB), s.Configuration)
	_, result := test.UpdateCommentsOK(s.T(), svcWithCollaborator1.Context, svcWithCollaborator1, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *collaborator1, updatedBody, markdownMarkup)

	// update comment by collaborator2
	updatedBody = "Modified body of comment"
	updateCommentPayload = newUpdateCommentsPayload(updatedBody, &markdownMarkup)
	svcWithCollaborator2 := testsupport.ServiceAsSpaceUser("Comments-Service", wittoken.NewManagerWithPrivateKey(priv), *collaborator2, &TestSpaceAuthzService{*collaborator2})
	commentCtrl = NewCommentsController(svcWithCollaborator2, gormapplication.NewGormDB(s.DB), s.Configuration)
	_, result = test.UpdateCommentsOK(s.T(), svcWithCollaborator2.Context, svcWithCollaborator2, commentCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, *collaborator1, updatedBody, markdownMarkup)
}

func (s *CommentsSuite) TestNotificationSendOnUpdate() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assert.True(s.T(), len(s.notification.Messages) > 0)
	assert.Equal(s.T(), "comment.update", s.notification.Messages[0].MessageType)
	assert.Equal(s.T(), c.Data.ID.String(), s.notification.Messages[0].TargetID)
}
