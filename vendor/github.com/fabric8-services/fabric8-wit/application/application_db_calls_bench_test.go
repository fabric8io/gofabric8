package application_test

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

type Space struct {
	gorm.Model
	gormsupport.Lifecycle
	ID          uuid.UUID
	Version     int
	Name        string
	Description string
	OwnerId     uuid.UUID `sql:"type:uuid"` // Belongs To Identity
}

type BenchDbOperations struct {
	gormbench.DBBenchSuite
	clean func()
	repo  space.Repository
	ctx   context.Context
	appDB application.DB
	dbPq  *sql.DB
}

func BenchmarkRunDbOperations(b *testing.B) {
	testsupport.Run(b, &BenchDbOperations{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *BenchDbOperations) SetupSuite() {
	s.DBBenchSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBBenchSuite.PopulateDBBenchSuite(s.ctx)
	var err error
	s.dbPq, err = sql.Open("postgres", "host=localhost port=5432 user=postgres password=mysecretpassword dbname=postgres sslmode=disable connect_timeout=5")
	if err != nil {
		s.B().Fail()
	}
	s.dbPq.SetMaxOpenConns(10)
	s.dbPq.SetMaxIdleConns(10)
	s.dbPq.SetConnMaxLifetime(0)
}

func (s *BenchDbOperations) SetupBenchmark() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = space.NewRepository(s.DB)
	s.appDB = gormapplication.NewGormDB(s.DB)
}

func (s *BenchDbOperations) TearDownBenchmark() {
	s.clean()
}

func (s *BenchDbOperations) BenchmarkPqSelectOneQuery() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	todo := func() {
		result, err := s.dbPq.Query("SELECT 1")
		defer result.Close()
		if err != nil {
			s.B().Fail()
		}
		for result.Next() {
		}
	}
	for n := 0; n < s.B().N; n++ {
		todo()
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectOneQuery() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	todo := func() {
		result, err := s.DB.Raw("select 1").Rows()
		defer result.Close()
		if err != nil {
			s.B().Fail()
		}
		for result.Next() {
		}
	}
	for n := 0; n < s.B().N; n++ {
		todo()
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceNameFirst() {
	var sp Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		db := s.DB.Select("name")
		db.Where("id=?", space.SystemSpace).First(&sp)
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceNameFind() {
	var sp Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		db := s.DB.Table("spaces").Select("name")
		db.Where("id=?", space.SystemSpace).Find(&sp)
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceNameRaw() {
	s.B().ResetTimer()
	s.B().ReportAllocs()
	todo := func() {
		var names []string
		result, err := s.DB.Raw("select name from spaces where id=?", space.SystemSpace).Rows()
		if err != nil {
			s.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var wit string
			result.Scan(&wit)
			names = append(names, wit)
		}
	}
	for n := 0; n < s.B().N; n++ {
		todo()
	}
}

func (s *BenchDbOperations) BenchmarkPqSelectSpaceNamePreparedStatement() {
	queryStmt, err := s.dbPq.Prepare("SELECT name FROM spaces WHERE id=$1")
	if err != nil {
		s.B().Fail()
	}
	var sp space.Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		err = queryStmt.QueryRow(space.SystemSpace).Scan(&sp.Name)
		if err != nil {
			s.B().Fail()
		}
	}
}

func (s *BenchDbOperations) BenchmarkPqSelectSpaceNameQueryRow() {
	var sp space.Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		err := s.dbPq.QueryRow("SELECT name FROM spaces WHERE id=$1", space.SystemSpace).Scan(&sp.Name)
		if err != nil {
			s.B().Fail()
		}
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceFirst() {
	var sp Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		db := s.DB.Select("version, name, description, owner_id")
		db.Where("id=?", space.SystemSpace).First(&sp)
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceFind() {
	var sp Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		db := s.DB.Table("spaces").Select("version, name, description, owner_id")
		db.Where("id=?", space.SystemSpace).Find(&sp)
	}
}

func (s *BenchDbOperations) BenchmarkGormSelectSpaceRaw() {
	s.B().ResetTimer()
	s.B().ReportAllocs()

	todo := func() {
		var sps []space.Space
		result, err := s.DB.Raw("select version, name, description, owner_id from spaces where id=?", space.SystemSpace).Rows()
		if err != nil {
			s.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var sp space.Space
			result.Scan(
				&sp.Version,
				&sp.Name,
				&sp.Description,
				&sp.OwnerId)
			sps = append(sps, sp)
		}
	}
	for n := 0; n < s.B().N; n++ {
		todo()
	}
}

func (s *BenchDbOperations) BenchmarkPqSelectSpacePreparedStatement() {
	queryStmt, err := s.dbPq.Prepare("SELECT version, name, description, owner_id FROM spaces WHERE id=$1")
	if err != nil {
		s.B().Fail()
	}
	var sp space.Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		err = queryStmt.QueryRow(space.SystemSpace).Scan(
			&sp.Version,
			&sp.Name,
			&sp.Description,
			&sp.OwnerId)
		if err != nil {
			s.B().Logf("%v", err)
			s.B().Fail()
		}
	}
}

func (s *BenchDbOperations) BenchmarkPqSelectSpaceQueryRow() {
	var sp space.Space
	s.B().ResetTimer()
	s.B().ReportAllocs()
	for n := 0; n < s.B().N; n++ {
		err := s.dbPq.QueryRow("SELECT version, name, description, owner_id FROM spaces WHERE id=$1", space.SystemSpace).Scan(
			&sp.Version,
			&sp.Name,
			&sp.Description,
			&sp.OwnerId)
		if err != nil {
			s.B().Logf("%v", err)
			s.B().Fail()
		}
	}
}
