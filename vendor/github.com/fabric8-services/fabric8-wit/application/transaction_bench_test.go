package application_test

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
)

type BenchTransactional struct {
	gormbench.DBBenchSuite
	clean func()
	repo  space.Repository
	ctx   context.Context
	appDB application.DB
	dbPq  *sql.DB
}

func BenchmarkRunTransactional(b *testing.B) {
	testsupport.Run(b, &BenchTransactional{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *BenchTransactional) SetupSuite() {
	s.DBBenchSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBBenchSuite.PopulateDBBenchSuite(s.ctx)
}

func (s *BenchTransactional) SetupBenchmark() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = space.NewRepository(s.DB)
	s.appDB = gormapplication.NewGormDB(s.DB)
}

func (s *BenchTransactional) TearDownBenchmark() {
	s.clean()
}

func (s *BenchTransactional) transactionLoadSpace() {
	err := application.Transactional(s.appDB, func(appl application.Application) error {
		_, err := s.repo.Load(s.ctx, space.SystemSpace)
		return err
	})
	if err != nil {
		s.B().Fail()
	}
}

func (s *BenchTransactional) BenchmarkApplTransaction() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		s.transactionLoadSpace()
	}
}
