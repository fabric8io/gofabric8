package comment_test

import (
	"testing"
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestCommentRepository struct {
	gormtestsupport.DBTestSuite
	clean        func()
	testIdentity account.Identity
	repo         comment.Repository
	ctx          context.Context
}

func TestRunCommentRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestCommentRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *TestCommentRepository) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *TestCommentRepository) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = comment.NewRepository(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity
}

func (s *TestCommentRepository) TearDownTest() {
	s.clean()
}

func newComment(parentID uuid.UUID, body, markup string) *comment.Comment {
	return &comment.Comment{
		ParentID: parentID,
		Body:     body,
		Markup:   markup,
		Creator:  uuid.NewV4(),
	}
}

func (s *TestCommentRepository) createComment(c *comment.Comment, creator uuid.UUID) {
	err := s.repo.Create(s.ctx, c, creator)
	require.Nil(s.T(), err)
}

func (s *TestCommentRepository) createComments(comments []*comment.Comment, creator uuid.UUID) {
	for _, c := range comments {
		s.createComment(c, creator)
	}
}

func (s *TestCommentRepository) TestCreateCommentWithMarkup() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	// when
	s.repo.Create(s.ctx, comment, s.testIdentity.ID)
	// then
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(s.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(s.T(), comment.CreatedAt.After(time.Now()), "Comment was not created, CreatedAt after Now()?")
}

func (s *TestCommentRepository) TestCreateCommentWithoutMarkup() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", "")
	// when
	s.repo.Create(s.ctx, comment, s.testIdentity.ID)
	// then
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	require.NotNil(s.T(), comment.CreatedAt, "Comment was not created?")
	assert.False(s.T(), comment.CreatedAt.After(time.Now()), "CreatedAt after Now()?")
	assert.Equal(s.T(), rendering.SystemMarkupDefault, comment.Markup)
}

func (s *TestCommentRepository) TestSaveCommentWithMarkup() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupPlainText)
	s.createComment(comment, s.testIdentity.ID)
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	// when
	comment.Body = "Test AB"
	comment.Markup = rendering.SystemMarkupMarkdown
	s.repo.Save(s.ctx, comment, s.testIdentity.ID)
	offset := 0
	limit := 1
	comments, _, err := s.repo.List(s.ctx, comment.ParentID, &offset, &limit)
	// then
	require.Nil(s.T(), err)
	require.Equal(s.T(), 1, len(comments), "List returned more then expected based on parentID")
	assert.Equal(s.T(), "Test AB", comments[0].Body)
	assert.Equal(s.T(), rendering.SystemMarkupMarkdown, comments[0].Markup)
}

func (s *TestCommentRepository) TestSaveCommentWithoutMarkup() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	s.createComment(comment, s.testIdentity.ID)
	assert.NotNil(s.T(), comment.ID, "Comment was not created, ID nil")
	// when
	comment.Body = "Test AB"
	comment.Markup = ""
	s.repo.Save(s.ctx, comment, s.testIdentity.ID)
	offset := 0
	limit := 1
	comments, _, err := s.repo.List(s.ctx, comment.ParentID, &offset, &limit)
	// then
	require.Nil(s.T(), err)
	require.Equal(s.T(), 1, len(comments), "List returned more then expected based on parentID")
	assert.Equal(s.T(), "Test AB", comments[0].Body)
	assert.Equal(s.T(), rendering.SystemMarkupPlainText, comments[0].Markup)
}

func (s *TestCommentRepository) TestDeleteComment() {
	// given
	parentID := uuid.NewV4()
	c := &comment.Comment{
		ParentID: parentID,
		Body:     "Test AA",
		Creator:  uuid.NewV4(),
		ID:       uuid.NewV4(),
	}
	s.repo.Create(s.ctx, c, s.testIdentity.ID)
	require.NotEqual(s.T(), uuid.Nil, c.ID)
	// when
	err := s.repo.Delete(s.ctx, c.ID, s.testIdentity.ID)
	// then
	assert.Nil(s.T(), err)
}

func (s *TestCommentRepository) TestCountComments() {
	// given
	parentID := uuid.NewV4()
	comment1 := newComment(parentID, "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment(uuid.NewV4(), "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	s.createComments(comments, s.testIdentity.ID)
	// when
	count, err := s.repo.Count(s.ctx, parentID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), 1, count)
}

func (s *TestCommentRepository) TestListComments() {
	// given
	comment1 := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment(uuid.NewV4(), "Test B", rendering.SystemMarkupMarkdown)
	createdComments := []*comment.Comment{comment1, comment2}
	s.createComments(createdComments, s.testIdentity.ID)
	// when
	offset := 0
	limit := 1
	resultComments, _, err := s.repo.List(s.ctx, comment1.ParentID, &offset, &limit)
	// then
	require.Nil(s.T(), err)
	require.Equal(s.T(), 1, len(resultComments))
	assert.Equal(s.T(), comment1.Body, resultComments[0].Body)
}

func (s *TestCommentRepository) TestListCommentsWrongOffset() {
	// given
	comment1 := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment(uuid.NewV4(), "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	s.createComments(comments, s.testIdentity.ID)
	// when
	offset := -1
	limit := 1
	_, _, err := s.repo.List(s.ctx, comment1.ParentID, &offset, &limit)
	// then
	assert.NotNil(s.T(), err)
}

func (s *TestCommentRepository) TestListCommentsWrongLimit() {
	// given
	comment1 := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment(uuid.NewV4(), "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	s.createComments(comments, s.testIdentity.ID)
	// when
	offset := 0
	limit := -1
	_, _, err := s.repo.List(s.ctx, comment1.ParentID, &offset, &limit)
	// then
	assert.NotNil(s.T(), err)
}

func (s *TestCommentRepository) TestLoadComment() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	s.createComment(comment, s.testIdentity.ID)
	// when
	loadedComment, err := s.repo.Load(s.ctx, comment.ID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), comment.ID, loadedComment.ID)
	assert.Equal(s.T(), comment.Body, loadedComment.Body)
}

func (s *TestCommentRepository) TestExistsComment() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("comment exists", func(t *testing.T) {
		// given
		comment := newComment(uuid.NewV4(), "Test C", rendering.SystemMarkupMarkdown)
		s.createComment(comment, s.testIdentity.ID)
		// when
		err := s.repo.CheckExists(s.ctx, comment.ID.String())
		// then
		require.Nil(t, err)
	})

	t.Run("comment doesn't exist", func(t *testing.T) {
		// when
		err := s.repo.CheckExists(s.ctx, uuid.NewV4().String())
		// then

		require.IsType(t, errors.NotFoundError{}, err)
	})

}
