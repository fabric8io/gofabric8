package comment_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/comment"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

type BenchCommentRepository struct {
	gormbench.DBBenchSuite
	clean        func()
	testIdentity account.Identity
	repo         comment.Repository
	ctx          context.Context
}

func BenchmarkRunCommentRepository(b *testing.B) {
	testsupport.Run(b, &BenchCommentRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *BenchCommentRepository) SetupSuite() {
	s.DBBenchSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBBenchSuite.PopulateDBBenchSuite(s.ctx)
}

func (s *BenchCommentRepository) SetupBenchmark() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = comment.NewRepository(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	if err != nil {
		s.B().Fail()
	}
	s.testIdentity = *testIdentity
}

func (s *BenchCommentRepository) TearDownBenchmark() {
	s.clean()
}

func (s *BenchCommentRepository) createComment(c *comment.Comment, creator uuid.UUID) {
	err := s.repo.Create(s.ctx, c, creator)
	require.Nil(s.B(), err)
}

func (s *BenchCommentRepository) createComments(comments []*comment.Comment, creator uuid.UUID) {
	for _, c := range comments {
		s.createComment(c, creator)
	}
}

func (s *BenchCommentRepository) BenchmarkCreateCommentWithMarkup() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	// when
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		if err := s.repo.Create(s.ctx, comment, s.testIdentity.ID); err != nil {
			s.B().Fail()
		}
	}
}

func (s *BenchCommentRepository) BenchmarkLoadComment() {
	// given
	comment := newComment(uuid.NewV4(), "Test A", rendering.SystemMarkupMarkdown)
	s.createComment(comment, s.testIdentity.ID)
	// when
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		if loadedComment, err := s.repo.Load(s.ctx, comment.ID); err != nil || (err == nil && loadedComment == nil) {
			s.B().Fail()
		}
	}
}

func (s *BenchCommentRepository) BenchmarkCountComments() {
	// given
	parentID := uuid.NewV4()
	comment1 := newComment(parentID, "Test A", rendering.SystemMarkupMarkdown)
	comment2 := newComment(uuid.NewV4(), "Test B", rendering.SystemMarkupMarkdown)
	comments := []*comment.Comment{comment1, comment2}
	s.createComments(comments, s.testIdentity.ID)
	// when
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		if count, err := s.repo.Count(s.ctx, parentID); err != nil || (err == nil && count == 0) {
			s.B().Fail()
		}
	}
}

func (s *BenchCommentRepository) BenchmarkCreateDeleteComment() {
	// given
	parentID := uuid.NewV4()
	// when
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		c := &comment.Comment{
			ParentID: parentID,
			Body:     "Test AA" + uuid.NewV4().String(),
			Creator:  uuid.NewV4(),
			ID:       uuid.NewV4(),
		}
		if err := s.repo.Create(s.ctx, c, s.testIdentity.ID); err != nil {
			s.B().Fail()
		}
		if err := s.repo.Delete(s.ctx, c.ID, s.testIdentity.ID); err != nil {
			s.B().Fail()
		}
	}
}
