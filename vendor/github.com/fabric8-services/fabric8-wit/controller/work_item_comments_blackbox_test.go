package controller_test

import (
	"html"
	"net/http"
	"net/url"
	"testing"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/comment"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentREST struct {
	gormtestsupport.DBTestSuite
	db           *gormapplication.GormDB
	clean        func()
	testIdentity account.Identity
	ctx          context.Context
	notification testsupport.NotificationChannel
}

func TestRunCommentREST(t *testing.T) {
	suite.Run(t, &TestCommentREST{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (rest *TestCommentREST) SetupTest() {
	resource.Require(rest.T(), resource.Database)
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	testIdentity, err := testsupport.CreateTestIdentity(rest.DB, "TestCommentREST setup user", "test provider")
	require.Nil(rest.T(), err)
	rest.testIdentity = *testIdentity
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	rest.ctx = goa.NewContext(context.Background(), nil, req, params)
	rest.notification = testsupport.NotificationChannel{}
}

func (rest *TestCommentREST) TearDownTest() {
	rest.clean()
}

func (rest *TestCommentREST) SecuredController() (*goa.Service, *WorkItemCommentsController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("WorkItemComment-Service", wittoken.NewManagerWithPrivateKey(priv), rest.testIdentity)
	return svc, NewNotifyingWorkItemCommentsController(svc, rest.db, &rest.notification, rest.Configuration)
}

func (rest *TestCommentREST) UnSecuredController() (*goa.Service, *WorkItemCommentsController) {
	svc := goa.New("WorkItemComment-Service")
	return svc, NewWorkItemCommentsController(svc, rest.db, rest.Configuration)
}

func (rest *TestCommentREST) newCreateWorkItemCommentsPayload(body string, markup *string) *app.CreateWorkItemCommentsPayload {
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

func (rest *TestCommentREST) createDefaultWorkItem() *workitem.WorkItem {
	rest.T().Log("Creating work item with modifier ID:", rest.testIdentity.ID)
	var workItem *workitem.WorkItem
	err := application.Transactional(rest.db, func(appl application.Application) error {
		repo := appl.WorkItems()
		wi, err := repo.Create(
			rest.ctx,
			space.SystemSpace,
			workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle: "A",
				workitem.SystemState: "new",
			},
			rest.testIdentity.ID)
		if err != nil {
			return errors.WithStack(err)
		}
		workItem = wi
		return nil
	})
	require.Nil(rest.T(), err)
	return workItem
}

func assertWorkItemComment(t *testing.T, c *app.Comment, expectedBody string, expectedMarkup string) {
	assert.NotNil(t, c)
	assert.Equal(t, "comments", c.Type)
	assert.NotNil(t, c.ID)
	require.NotNil(t, c.Attributes)
	assert.Equal(t, expectedBody, *c.Attributes.Body)
	assert.Equal(t, expectedMarkup, *c.Attributes.Markup)
	assert.Equal(t, rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *c.Attributes.BodyRendered)
	require.NotNil(t, c.Attributes.CreatedAt)
	assert.WithinDuration(t, time.Now(), *c.Attributes.CreatedAt, 2*time.Second)
	require.NotNil(t, c.Relationships)
	require.NotNil(t, c.Relationships.Creator)
	require.NotNil(t, c.Relationships.Creator.Data)
	assert.Equal(t, "identities", c.Relationships.Creator.Data.Type)
	assert.NotNil(t, c.Relationships.Creator.Data.ID)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithMarkup() {
	// given
	wi := rest.createDefaultWorkItem()

	// when
	markup := rendering.SystemMarkupMarkdown
	p := rest.newCreateWorkItemCommentsPayload("Test", &markup)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assertComment(rest.T(), c.Data, rest.testIdentity, "Test", markup)
}

func (rest *TestCommentREST) TestSuccessCreateSingleCommentWithDefaultMarkup() {
	// given
	wi := rest.createDefaultWorkItem()
	// when
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assertComment(rest.T(), c.Data, rest.testIdentity, "Test", rendering.SystemMarkupDefault)
}

func (rest *TestCommentREST) TestNotificationSendOnCreate() {
	// given
	wi := rest.createDefaultWorkItem()
	// when
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.SecuredController()
	_, c := test.CreateWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
	// then
	assert.True(rest.T(), len(rest.notification.Messages) > 0)
	assert.Equal(rest.T(), "comment.create", rest.notification.Messages[0].MessageType)
	assert.Equal(rest.T(), c.Data.ID.String(), rest.notification.Messages[0].TargetID)
}

func (rest *TestCommentREST) setupComments() (workitem.WorkItem, []*comment.Comment) {
	wi := rest.createDefaultWorkItem()
	comments := make([]*comment.Comment, 4)
	comments[0] = &comment.Comment{ParentID: wi.ID, Body: "Test 1", Creator: rest.testIdentity.ID}
	comments[1] = &comment.Comment{ParentID: wi.ID, Body: "Test 2", Creator: rest.testIdentity.ID}
	comments[2] = &comment.Comment{ParentID: wi.ID, Body: "Test 3", Creator: rest.testIdentity.ID}
	comments[3] = &comment.Comment{ParentID: uuid.NewV4(), Body: "Test 1", Creator: rest.testIdentity.ID}
	application.Transactional(rest.db, func(app application.Application) error {
		repo := app.Comments()
		for _, c := range comments {
			repo.Create(rest.ctx, c, rest.testIdentity.ID)
		}
		return nil
	})
	return *wi, comments
}

func assertComments(t *testing.T, expectedIdentity account.Identity, comments *app.CommentList) {
	require.Equal(t, 3, len(comments.Data))
	assertComment(t, comments.Data[0], expectedIdentity, "Test 3", rendering.SystemMarkupDefault) // items are returned in reverse order or creation
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOK() {
	// given
	wi, _ := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, nil)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOKUsingExpiredIfModifiedSinceHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifModifiedSince := app.ToHTTPTime(comments[3].UpdatedAt.Add(-1 * time.Hour))
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemOKUsingExpiredIfNoneMatchHeader() {
	// given
	wi, _ := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifNoneMatch := "foo"
	res, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertComments(rest.T(), rest.testIdentity, cs)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemNotModifiedUsingIfModifiedSinceHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifModifiedSince := app.ToHTTPTime(comments[3].UpdatedAt)
	res := test.ListWorkItemCommentsNotModified(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestListCommentsByParentWorkItemNotModifiedUsingIfNoneMatchHeader() {
	// given
	wi, comments := rest.setupComments()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 3
	ifNoneMatch := app.GenerateEntitiesTag([]app.ConditionalRequestEntity{
		comments[2],
		comments[1],
		comments[0],
	})
	res := test.ListWorkItemCommentsNotModified(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestCommentREST) TestEmptyListCommentsByParentWorkItem() {
	// given
	wi := rest.createDefaultWorkItem()
	// when
	svc, ctrl := rest.UnSecuredController()
	offset := "0"
	limit := 1
	_, cs := test.ListWorkItemCommentsOK(rest.T(), svc.Context, svc, ctrl, wi.ID, &limit, &offset, nil, nil)
	// then
	assert.Equal(rest.T(), 0, len(cs.Data))
}

func (rest *TestCommentREST) TestCreateSingleCommentMissingWorkItem() {
	// given
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	// when/then
	svc, ctrl := rest.SecuredController()
	test.CreateWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), p)
}

func (rest *TestCommentREST) TestCreateSingleNoAuthorized() {
	// given
	wi := rest.createDefaultWorkItem()
	// when/then
	p := rest.newCreateWorkItemCommentsPayload("Test", nil)
	svc, ctrl := rest.UnSecuredController()
	test.CreateWorkItemCommentsUnauthorized(rest.T(), svc.Context, svc, ctrl, wi.ID, p)
}

// Can not be tested via normal Goa testing framework as setting empty body on CreateCommentAttributes is
// validated before Request to service is made. Validate model and assume it will be validated before hitting
// service when mounted. Test to show intent.
func (rest *TestCommentREST) TestShouldNotCreateEmptyBody() {
	// given
	p := rest.newCreateWorkItemCommentsPayload("", nil)
	// when
	err := p.Validate()
	// then
	require.NotNil(rest.T(), err)
}

func (rest *TestCommentREST) TestListCommentsByMissingParentWorkItem() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	offset := "0"
	limit := 1
	test.ListWorkItemCommentsNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4(), &limit, &offset, nil, nil)
}
