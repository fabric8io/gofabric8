package area_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/test"

	uuid "github.com/satori/go.uuid"
)

type BenchAreaRepository struct {
	gormbench.DBBenchSuite
	repo      area.Repository
	repoSpace space.Repository
	clean     func()
}

func BenchmarkRunAreaRepository(b *testing.B) {
	resource.Require(b, resource.Database)
	test.Run(b, &BenchAreaRepository{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

func (bench *BenchAreaRepository) SetupBenchmark() {
	bench.clean = cleaner.DeleteCreatedEntities(bench.DB)
	bench.repo = area.NewAreaRepository(bench.DB)
	bench.repoSpace = space.NewRepository(bench.DB)
}

func (bench *BenchAreaRepository) TearDownBenchmark() {
	bench.clean()
}

func (bench *BenchAreaRepository) BenchmarkRootArea() {
	if err := bench.repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space.SystemSpace,
	}); err != nil {
		bench.B().Fail()
	}
	// when
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if _, err := bench.repo.Root(context.Background(), space.SystemSpace); err != nil {
			bench.B().Fail()
		}
	}
}

func (bench *BenchAreaRepository) BenchmarkCreateArea() {
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		newSpace := space.Space{
			Name: "BenchmarkCreateArea " + uuid.NewV4().String(),
		}
		s, err := bench.repoSpace.Create(context.Background(), &newSpace)
		if err != nil {
			bench.B().Fail()
		}
		a := area.Area{
			Name:    "TestCreateArea",
			SpaceID: s.ID,
		}
		if err := bench.repo.Create(context.Background(), &a); err != nil {
			bench.B().Fail()
		}
	}
}

func (bench *BenchAreaRepository) BenchmarkListAreaBySpace() {
	if err := bench.repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space.SystemSpace,
	}); err != nil {
		bench.B().Fail()
	}
	// when
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if its, err := bench.repo.List(context.Background(), space.SystemSpace); err != nil || (err == nil && len(its) == 0) {
			bench.B().Fail()
		}
	}
}
