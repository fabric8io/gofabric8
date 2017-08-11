package remoteworkitem

import (
	"net/http"
	"net/url"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/application"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestTrackerQueryRepository struct {
	gormtestsupport.DBTestSuite

	trackerRepo application.TrackerRepository
	queryRepo   application.TrackerQueryRepository

	clean func()
}

func TestRunTrackerQueryRepository(t *testing.T) {
	suite.Run(t, &TestTrackerQueryRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (test *TestTrackerQueryRepository) SetupTest() {
	test.trackerRepo = NewTrackerRepository(test.DB)
	test.queryRepo = NewTrackerQueryRepository(test.DB)

	test.clean = cleaner.DeleteCreatedEntities(test.DB)
}

func (test *TestTrackerQueryRepository) TearDownTest() {
	test.clean()
}

func (test *TestTrackerQueryRepository) TestTrackerQueryCreate() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	query, err := test.queryRepo.Create(ctx, "abc", "xyz", "lmn", space.SystemSpace)
	assert.IsType(t, NotFoundError{}, err)
	assert.Nil(t, query)

	tracker, err := test.trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
	query, err = test.queryRepo.Create(ctx, "abc", "xyz", tracker.ID, space.SystemSpace)
	assert.Nil(t, err)
	assert.Equal(t, "abc", query.Query)
	assert.Equal(t, "xyz", query.Schedule)

	query2, err := test.queryRepo.Load(ctx, query.ID)
	assert.Nil(t, err)
	assert.Equal(t, query, query2)
}

func (test *TestTrackerQueryRepository) TestExistsTrackerQuery() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("tracker query exists", func(t *testing.T) {
		t.Parallel()
		// given
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		tracker, err := test.trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
		assert.Nil(t, err)

		query, err := test.queryRepo.Create(ctx, "abc", "xyz", tracker.ID, space.SystemSpace)
		assert.Nil(t, err)

		err = test.queryRepo.CheckExists(ctx, query.ID)
		assert.Nil(t, err)
	})

	t.Run("tracker query doesn't exist", func(t *testing.T) {
		t.Parallel()
		req := &http.Request{Host: "localhost"}
		params := url.Values{}
		ctx := goa.NewContext(context.Background(), nil, req, params)

		err := test.queryRepo.CheckExists(ctx, "11111111111")
		require.IsType(t, errs.NotFoundError{}, err)
	})

}

func (test *TestTrackerQueryRepository) TestTrackerQuerySave() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	query, err := test.queryRepo.Load(ctx, "abcd")
	assert.IsType(t, NotFoundError{}, err)
	assert.Nil(t, query)

	tracker, err := test.trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
	tracker2, err := test.trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
	query, err = test.queryRepo.Create(ctx, "abc", "xyz", tracker.ID, space.SystemSpace)
	query2, err := test.queryRepo.Load(ctx, query.ID)
	assert.Nil(t, err)
	assert.Equal(t, query, query2)

	query.Query = "after"
	query.Schedule = "the"
	query.TrackerID = tracker2.ID
	if err != nil {
		t.Errorf("could not convert id: %s", tracker2.ID)
	}

	query2, err = test.queryRepo.Save(ctx, *query)
	assert.Nil(t, err)
	assert.Equal(t, query, query2)

	test.trackerRepo.Delete(ctx, "10000")

	query.TrackerID = "10000"
	query2, err = test.queryRepo.Save(ctx, *query)
	assert.IsType(t, NotFoundError{}, err)
	assert.Nil(t, query2)
}

func (test *TestTrackerQueryRepository) TestTrackerQueryDelete() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	err := test.queryRepo.Delete(ctx, "asdf")
	assert.IsType(t, NotFoundError{}, err)

	tracker, _ := test.trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
	tq, _ := test.queryRepo.Create(ctx, "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker.ID, space.SystemSpace)
	err = test.queryRepo.Delete(ctx, tq.ID)
	assert.Nil(t, err)

	tq, err = test.queryRepo.Load(ctx, tq.ID)
	assert.IsType(t, NotFoundError{}, err)
	assert.Nil(t, tq)

	tq, err = test.queryRepo.Load(ctx, "100000")
	assert.IsType(t, NotFoundError{}, err)
	assert.Nil(t, tq)
}

func (test *TestTrackerQueryRepository) TestTrackerQueryList() {
	t := test.T()
	resource.Require(t, resource.Database)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	trackerqueries1, _ := test.queryRepo.List(ctx)

	tracker1, _ := test.trackerRepo.Create(ctx, "http://api.github.com", ProviderGithub)
	test.queryRepo.Create(ctx, "is:open is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker1.ID, space.SystemSpace)
	test.queryRepo.Create(ctx, "is:close is:issue user:arquillian author:aslakknutsen", "15 * * * * *", tracker1.ID, space.SystemSpace)

	tracker2, _ := test.trackerRepo.Create(ctx, "http://issues.jboss.com", ProviderJira)
	test.queryRepo.Create(ctx, "project = ARQ AND text ~ 'arquillian'", "15 * * * * *", tracker2.ID, space.SystemSpace)
	test.queryRepo.Create(ctx, "project = ARQ AND text ~ 'javadoc'", "15 * * * * *", tracker2.ID, space.SystemSpace)

	trackerqueries2, _ := test.queryRepo.List(ctx)
	assert.Equal(t, len(trackerqueries1)+4, len(trackerqueries2))
	trackerqueries3, _ := test.queryRepo.List(ctx)
	assert.Equal(t, trackerqueries2[1], trackerqueries3[1])
}
