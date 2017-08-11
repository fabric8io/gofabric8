package search_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"context"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestRunSearchRepositoryBlackboxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &searchRepositoryBlackboxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type searchRepositoryBlackboxTest struct {
	gormtestsupport.DBTestSuite
	modifierID uuid.UUID
	clean      func()
	searchRepo *search.GormSearchRepository
	wiRepo     *workitem.GormWorkItemRepository
	witRepo    *workitem.GormWorkItemTypeRepository
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
func (s *searchRepositoryBlackboxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

func (s *searchRepositoryBlackboxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe", "test")
	require.Nil(s.T(), err)
	s.modifierID = testIdentity.ID
	s.witRepo = workitem.NewWorkItemTypeRepository(s.DB)
	s.wiRepo = workitem.NewWorkItemRepository(s.DB)
	s.searchRepo = search.NewGormSearchRepository(s.DB)
}

func (s *searchRepositoryBlackboxTest) TearDownTest() {
	s.clean()
}

func (s *searchRepositoryBlackboxTest) TestRestrictByType() {
	// given
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)

	res, count, err := s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		s.wiRepo.Delete(ctx, wi.ID, s.modifierID)
	}

	extended := workitem.SystemBug
	base, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &extended, "base", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), base)
	require.NotNil(s.T(), base.ID)

	sub1, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &base.ID, "sub1", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub1)
	require.NotNil(s.T(), sub1.ID)

	sub2, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &base.ID, "subtwo", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub2)
	require.NotNil(s.T(), sub2.ID)

	wi1, err := s.wiRepo.Create(ctx, space.SystemSpace, sub1.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi1)

	wi2, err := s.wiRepo.Create(ctx, space.SystemSpace, sub2.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType 2",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi2)

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType", nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi1.ID, res[0].ID)
		assert.Equal(s.T(), res[0].Fields["system.order"], wi1.Fields["system.order"])
	}

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(1), count)
	if count == 1 {
		assert.Equal(s.T(), wi2.ID, res[0].ID)
		assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	}

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+sub2.ID.String()+" type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	res, count, err = s.searchRepo.SearchFullText(ctx, "TestRestrictByType type:"+base.ID.String()+" type:"+sub1.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), res[0].Fields["system.order"], wi2.Fields["system.order"])
	assert.Equal(s.T(), res[1].Fields["system.order"], wi1.Fields["system.order"])

	_, count, err = s.searchRepo.SearchFullText(ctx, "TRBTgorxi type:"+base.ID.String(), nil, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(0), count)
}

func (s *searchRepositoryBlackboxTest) TestFilterCount() {
	// given
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx := goa.NewContext(context.Background(), nil, req, params)
	notexistspace := "5f734617-472e-5dab-ab8d-e038345724b2"
	fs1 := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, notexistspace)
	res, count, err := s.searchRepo.Filter(ctx, fs1, nil, nil)
	require.Nil(s.T(), err)
	require.True(s.T(), count == uint64(len(res))) // safety check for many, many instances of bogus search results.
	for _, wi := range res {
		s.wiRepo.Delete(ctx, wi.ID, s.modifierID)
	}
	// when
	extended := workitem.SystemBug
	base, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &extended, "base", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), base)
	require.NotNil(s.T(), base.ID)

	sub1, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &base.ID, "sub1", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub1)
	require.NotNil(s.T(), sub1.ID)

	sub2, err := s.witRepo.Create(ctx, space.SystemSpace, nil, &base.ID, "subtwo", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), sub2)
	require.NotNil(s.T(), sub2.ID)

	wi1, err := s.wiRepo.Create(ctx, space.SystemSpace, sub1.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi1)

	wi2, err := s.wiRepo.Create(ctx, space.SystemSpace, sub2.ID, map[string]interface{}{
		workitem.SystemTitle: "Test TestRestrictByType 2",
		workitem.SystemState: "closed",
	}, s.modifierID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wi2)

	// then
	fs2 := fmt.Sprintf(`{"$AND": [{"space": "%s"}]}`, space.SystemSpace)
	start := 3
	res, count, err = s.searchRepo.Filter(ctx, fs2, &start, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), 0, len(res))

	res, count, err = s.searchRepo.Filter(ctx, fs2, nil, nil)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), uint64(2), count)
	assert.Equal(s.T(), 2, len(res))

}
