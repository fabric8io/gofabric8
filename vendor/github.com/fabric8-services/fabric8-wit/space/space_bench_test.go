package space_test

import (
	"context"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	gormbench "github.com/fabric8-services/fabric8-wit/gormtestsupport/benchmark"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/test"
	uuid "github.com/satori/go.uuid"
)

func BenchmarkRunRepoBBBench(b *testing.B) {
	test.Run(b, &repoSpaceBench{DBBenchSuite: gormbench.NewDBBenchSuite("../config.yaml")})
}

type repoSpaceBench struct {
	gormbench.DBBenchSuite
	repo  space.Repository
	clean func()
}

func (bench *repoSpaceBench) SetupBenchmark() {
	bench.repo = space.NewRepository(bench.DB)
	bench.clean = cleaner.DeleteCreatedEntities(bench.DB)
}

func (bench *repoSpaceBench) TearDownBenchmark() {
	bench.clean()
}

func (bench *repoSpaceBench) BenchmarkCreate() {
	// given
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		newSpace := space.Space{
			Name:    test.CreateRandomValidTestName("BenchmarkCreate"),
			OwnerId: uuid.Nil,
		}
		if s, err := bench.repo.Create(context.Background(), &newSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceByName() {
	newSpace := space.Space{
		Name:    test.CreateRandomValidTestName("BenchmarkLoadSpaceByName"),
		OwnerId: uuid.Nil,
	}
	if s, err := bench.repo.Create(context.Background(), &newSpace); err != nil || (err == nil && s == nil) {
		bench.B().Fail()
	}

	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.LoadByOwnerAndName(context.Background(), &newSpace.OwnerId, &newSpace.Name); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkLoadSpaceById() {
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, err := bench.repo.Load(context.Background(), space.SystemSpace); err != nil || (err == nil && s == nil) {
			bench.B().Fail()
		}
	}
}

func (bench *repoSpaceBench) BenchmarkList() {
	bench.B().ResetTimer()
	bench.B().ReportAllocs()
	for n := 0; n < bench.B().N; n++ {
		if s, _, err := bench.repo.List(context.Background(), nil, nil); err != nil || (err == nil && len(s) == 0) {
			bench.B().Fail()
		}
	}
}
