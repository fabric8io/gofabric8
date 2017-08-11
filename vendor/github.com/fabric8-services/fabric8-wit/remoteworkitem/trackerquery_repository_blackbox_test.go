package remoteworkitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/space"

	"context"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type trackerQueryRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	repo   application.TrackerQueryRepository
	trRepo application.TrackerRepository
	clean  func()
	ctx    context.Context
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *trackerQueryRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func TestRunTrackerQueryRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &trackerQueryRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *trackerQueryRepoBlackBoxTest) SetupTest() {
	s.repo = remoteworkitem.NewTrackerQueryRepository(s.DB)
	s.trRepo = remoteworkitem.NewTrackerRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *trackerQueryRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *trackerQueryRepoBlackBoxTest) TestFailDeleteZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		s.ctx,
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	_, err = s.repo.Create(
		s.ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID, space.SystemSpace)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	err = s.repo.Delete(s.ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFailSaveZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		s.ctx,
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	tq, err := s.repo.Create(
		s.ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID, space.SystemSpace)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}
	tq.ID = "0"

	_, err = s.repo.Save(s.ctx, *tq)
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}

func (s *trackerQueryRepoBlackBoxTest) TestFaiLoadZeroID() {
	// Create at least 1 item to avoid RowsEffectedCheck
	tr, err := s.trRepo.Create(
		s.ctx,
		"http://api.github.com",
		remoteworkitem.ProviderGithub)
	if err != nil {
		s.T().Error("Could not create tracker", err)
	}

	_, err = s.repo.Create(
		s.ctx,
		"project = ARQ AND text ~ 'arquillian'",
		"15 * * * * *",
		tr.ID, space.SystemSpace)
	if err != nil {
		s.T().Error("Could not create tracker query", err)
	}

	_, err = s.repo.Load(s.ctx, "0")
	require.IsType(s.T(), remoteworkitem.NotFoundError{}, err)
}
