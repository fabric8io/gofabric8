package auth_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"

	"context"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type stateBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo  auth.OauthStateReferenceRepository
	clean func()
	ctx   context.Context
}

func TestRunStateBlackBoxTest(t *testing.T) {
	suite.Run(t, &stateBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *stateBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func (s *stateBlackBoxTest) SetupTest() {
	s.repo = auth.NewOauthStateReferenceRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *stateBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *stateBlackBoxTest) TestCreateDeleteLoad() {
	// given
	state := &auth.OauthStateReference{
		ID:       uuid.NewV4(),
		Referrer: "domain.org"}

	state2 := &auth.OauthStateReference{
		ID:       uuid.NewV4(),
		Referrer: "anotherdomain.com"}

	_, err := s.repo.Create(s.ctx, state)
	require.Nil(s.T(), err, "Could not create state reference")
	_, err = s.repo.Create(s.ctx, state2)
	require.Nil(s.T(), err, "Could not create state reference")
	// when
	err = s.repo.Delete(s.ctx, state.ID)
	// then
	assert.Nil(s.T(), err)
	_, err = s.repo.Load(s.ctx, state.ID)
	require.NotNil(s.T(), err)
	require.IsType(s.T(), errors.NotFoundError{}, err)

	foundState, err := s.repo.Load(s.ctx, state2.ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), foundState)
	require.True(s.T(), state2.Equal(*foundState))
}
