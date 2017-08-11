package workitem_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
)

type BenchWorkItemTypeRepository struct {
	gormbench.DBBenchSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

func BenchmarkRunWorkItemTypeRepository(b *testing.B) {
	testsupport.Run(b, &BenchWorkItemTypeRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (s *BenchWorkItemTypeRepository) SetupSuite() {
	s.DBBenchSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBBenchSuite.PopulateDBBenchSuite(s.ctx)
}

func (s *BenchWorkItemTypeRepository) SetupBenchmark() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
}

func (s *BenchWorkItemTypeRepository) TearDownBenchmark() {
	s.clean()
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoad() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=? AND space_id=?", workitem.SystemExperience, space.SystemSpace).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoadTypeFromDB() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		res := workitem.WorkItemType{}
		db := r.DB.Model(&res).Where("id=?", workitem.SystemExperience).First(&res)
		if db.RecordNotFound() {
			r.B().Fail()
		}
		if err := db.Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkLoadWorkItemType() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, err := r.repo.Load(context.Background(), space.SystemSpace, workitem.SystemExperience); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListWorkItemTypes() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if s, err := r.repo.List(context.Background(), space.SystemSpace, nil, nil); err != nil || (err == nil && s == nil) {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListWorkItemTypesTransaction() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		if err := application.Transactional(gormapplication.NewGormDB(r.DB), func(app application.Application) error {
			_, err := r.repo.List(context.Background(), space.SystemSpace, nil, nil)
			return err
		}); err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListPlannerItems() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		path := path.Path{}
		db := r.DB.Select("id").Where("space_id = ? AND path::text LIKE '"+path.ConvertToLtree(workitem.SystemPlannerItem)+".%'", space.SystemSpace.String())

		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListFind() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		db := r.DB.Where("space_id = ?", space.SystemSpace)
		if err := db.Find(&rows).Error; err != nil {
			r.B().Fail()
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScan() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select  from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			wit := workitem.WorkItemType{}
			result.Scan(&wit)
			rows = append(rows, wit)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanAll() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.WorkItemType
		result, err := r.DB.Raw("select * from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			wit := workitem.WorkItemType{}
			result.Scan(&wit)
			rows = append(rows, wit)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanName() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []string
		result, err := r.DB.Raw("select name from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var witName string
			result.Scan(&witName)
			rows = append(rows, witName)
		}
	}
}

func (r *BenchWorkItemTypeRepository) BenchmarkListRawScanFields() {
	r.B().ResetTimer()
	r.B().ReportAllocs()
	for n := 0; n < r.B().N; n++ {
		var rows []workitem.FieldDefinition
		result, err := r.DB.Raw("select fields from work_item_types where space_id = ?", space.SystemSpace).Rows()
		if err != nil {
			r.B().Fail()
		}
		defer result.Close()
		for result.Next() {
			var field workitem.FieldDefinition
			result.Scan(&field)
			rows = append(rows, field)
		}
	}
}
