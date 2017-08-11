package workitem_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type workItemTypeRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

func TestRunWorkItemTypeRepoBlackBoxTest(t *testing.T) {
	suite.Run(t, &workItemTypeRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemTypeRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)

	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	s.ctx = goa.NewContext(context.Background(), nil, req, params)
}

func (s *workItemTypeRepoBlackBoxTest) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.repo = workitem.NewWorkItemTypeRepository(s.DB)
	workitem.ClearGlobalWorkItemTypeCache()
}

func (s *workItemTypeRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWIT() {

	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	// Test that we can create a WIT with the same name as before.
	wit3, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit3)
	require.NotNil(s.T(), wit3.ID)

	wit2, err := s.repo.Load(s.ctx, space.SystemSpace, wit.ID)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Fields)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), workitem.KindFloat, field.Type.GetKind())
	assert.Equal(s.T(), true, field.Required)
}

func (s *workItemTypeRepoBlackBoxTest) TestExistsWIT() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("wit exists", func(t *testing.T) {
		t.Parallel()
		// given
		wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
			"foo": {
				Required: true,
				Type:     &workitem.SimpleType{Kind: workitem.KindFloat},
			},
		})
		require.Nil(s.T(), err)
		require.NotNil(s.T(), wit)
		require.NotNil(s.T(), wit.ID)

		err = s.repo.CheckExists(s.ctx, wit.ID.String())
		require.Nil(s.T(), err)
	})

	t.Run("wit doesn't exist", func(t *testing.T) {
		t.Parallel()
		err := s.repo.CheckExists(s.ctx, uuid.NewV4().String())

		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (s *workItemTypeRepoBlackBoxTest) TestCreateLoadWITWithList() {
	wit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
		},
	})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit)
	require.NotNil(s.T(), wit.ID)

	wit3, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, "foo_bar", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), wit3)
	require.NotNil(s.T(), wit3.ID)

	wit2, err := s.repo.Load(s.ctx, space.SystemSpace, wit.ID)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), wit2)
	require.NotNil(s.T(), wit2.Fields)
	field := wit2.Fields["foo"]
	require.NotNil(s.T(), field)
	assert.Equal(s.T(), workitem.KindList, field.Type.GetKind())
	assert.Equal(s.T(), true, field.Required)
}

func (s *workItemTypeRepoBlackBoxTest) TestCreateWITWithBaseType() {
	basetype := "foo.bar"
	baseWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, nil, basetype, nil, "fa-bomb", map[string]workitem.FieldDefinition{
		"foo": {
			Required: true,
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindString}},
		},
	})

	require.Nil(s.T(), err)
	require.NotNil(s.T(), baseWit)
	require.NotNil(s.T(), baseWit.ID)
	extendedWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, &baseWit.ID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	require.Nil(s.T(), err)
	require.NotNil(s.T(), extendedWit)
	require.NotNil(s.T(), extendedWit.Fields)
	// the Field 'foo' must exist since it is inherited from the base work item type
	assert.NotNil(s.T(), extendedWit.Fields["foo"])
}

func (s *workItemTypeRepoBlackBoxTest) TestDoNotCreateWITWithMissingBaseType() {
	baseTypeID := uuid.Nil
	extendedWit, err := s.repo.Create(s.ctx, space.SystemSpace, nil, &baseTypeID, "foo.baz", nil, "fa-bomb", map[string]workitem.FieldDefinition{})
	// expect an error as the given base type does not exist
	require.NotNil(s.T(), err)
	require.Nil(s.T(), extendedWit)
}
