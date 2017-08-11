package area_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/area"
	errs "github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaRepository struct {
	gormtestsupport.DBTestSuite
	testIdentity account.Identity
	clean        func()
}

func TestRunAreaRepository(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &TestAreaRepository{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

func (s *TestAreaRepository) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "WorkItemSuite setup user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = *testIdentity
}

func (s *TestAreaRepository) TearDownTest() {
	s.clean()
}

func (s *TestAreaRepository) TestCreateAreaWithSameNameFail() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestCreateAreaWithSameNameFail"
	newSpace := space.Space{
		Name:    "Space 1 " + uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	repo.Create(context.Background(), &a)
	require.NotEqual(s.T(), uuid.Nil, a.ID)
	require.False(s.T(), a.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()")
	assert.Equal(s.T(), name, a.Name)
	anotherAreaWithSameName := area.Area{
		Name:    a.Name,
		SpaceID: space.ID,
	}
	// when
	err = repo.Create(context.Background(), &anotherAreaWithSameName)
	// then
	require.NotNil(s.T(), err)
	// In case of unique constrain error, a DataConflictError is returned.
	_, ok := errors.Cause(err).(errs.DataConflictError)
	assert.True(s.T(), ok)
}

func (s *TestAreaRepository) TestCreateArea() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestCreateArea"
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	// when
	err = repo.Create(context.Background(), &a)
	// then
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), uuid.Nil, a.ID)
	assert.True(s.T(), !a.CreatedAt.After(time.Now()), "Area was not created, CreatedAt after Now()?")
	assert.Equal(s.T(), name, a.Name)
}

func (s *TestAreaRepository) TestExistsArea() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("area exists", func(t *testing.T) {
		// given
		repo := area.NewAreaRepository(s.DB)
		name := "TestCreateArea"
		newSpace := space.Space{
			Name:    uuid.NewV4().String(),
			OwnerId: s.testIdentity.ID,
		}
		repoSpace := space.NewRepository(s.DB)
		space, err := repoSpace.Create(context.Background(), &newSpace)
		require.Nil(t, err)
		a := area.Area{
			Name:    name,
			SpaceID: space.ID,
		}
		// when
		err = repo.Create(context.Background(), &a)
		// then
		require.Nil(t, err)
		require.NotEqual(t, uuid.Nil, a.ID)

		// when
		err1 := repo.CheckExists(context.Background(), a.ID.String())
		// then
		require.Nil(t, err1)
	})

	t.Run("area doesn't exist", func(t *testing.T) {
		// given
		repo := area.NewAreaRepository(s.DB)
		// when
		err := repo.CheckExists(context.Background(), uuid.NewV4().String())
		// then
		require.IsType(t, errs.NotFoundError{}, err)
	})
}

func (s *TestAreaRepository) TestCreateChildArea() {
	// given
	repo := area.NewAreaRepository(s.DB)
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	name := "TestCreateChildArea"
	name2 := "TestCreateChildArea.1"
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(s.T(), err)
	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	// when
	err = repo.Create(context.Background(), &area2)
	// then
	require.Nil(s.T(), err)
	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(s.T(), err)
	require.NotNil(s.T(), actualArea)
	assert.Equal(s.T(), expectedPath, actualPath)
}

func (s *TestAreaRepository) TestGetAreaBySpaceIDAndNameAndPath() {
	t := s.T()

	resource.Require(t, resource.Database)

	repo := area.NewAreaRepository(s.DB)

	name := "space name " + uuid.NewV4().String()
	newSpace := space.Space{
		Name:    name,
		OwnerId: s.testIdentity.ID,
	}

	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(t, err)

	a := area.Area{
		Name:    name,
		SpaceID: space.ID,
		Path:    path.Path{},
	}
	err = repo.Create(context.Background(), &a)
	require.Nil(t, err)

	// So now we have a space and area with the same name.

	areaList, err := repo.Query(area.FilterBySpaceID(space.ID), area.FilterByPath(path.Path{}), area.FilterByName(name))
	require.Nil(t, err)

	// there must be ONLY 1 result, because of the space,name,path unique constraint
	require.Len(t, areaList, 1)

	rootArea := areaList[0]
	assert.Equal(t, name, rootArea.Name)
	assert.Equal(t, space.ID, rootArea.SpaceID)
}

func (s *TestAreaRepository) TestListAreaBySpace() {
	// given
	repo := area.NewAreaRepository(s.DB)
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space1, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)

	var createdAreaIds []uuid.UUID
	for i := 0; i < 3; i++ {
		name := "Test Area #20" + strconv.Itoa(i)

		a := area.Area{
			Name:    name,
			SpaceID: space1.ID,
		}
		err := repo.Create(context.Background(), &a)
		require.Nil(s.T(), err)
		createdAreaIds = append(createdAreaIds, a.ID)
		s.T().Log(a.ID)
	}
	newSpace2 := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	space2, err := repoSpace.Create(context.Background(), &newSpace2)
	require.Nil(s.T(), err)
	err = repo.Create(context.Background(), &area.Area{
		Name:    "Other Test area #20",
		SpaceID: space2.ID,
	})
	require.Nil(s.T(), err)
	// when
	its, err := repo.List(context.Background(), space1.ID)
	// then
	require.Nil(s.T(), err)
	require.Len(s.T(), its, 3)
	for i := 0; i < 3; i++ {
		assert.NotNil(s.T(), searchInAreaSlice(createdAreaIds[i], its))
	}
}

func searchInAreaSlice(searchKey uuid.UUID, areaList []area.Area) *area.Area {
	for i := 0; i < len(areaList); i++ {
		if searchKey == areaList[i].ID {
			return &areaList[i]
		}
	}
	return nil
}

func (s *TestAreaRepository) TestListChildrenOfParents() {
	// given
	resource.Require(s.T(), resource.Database)
	repo := area.NewAreaRepository(s.DB)
	name := "TestListChildrenOfParents"
	name2 := "TestListChildrenOfParents.1"
	name3 := "TestListChildrenOfParents.2"
	var createdAreaIDs []uuid.UUID
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	require.Nil(s.T(), err)
	// *** Create 1st child area ***
	// ltree field doesnt accept "-" , so we will save them as "_"
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(s.T(), err)
	createdAreaIDs = append(createdAreaIDs, area2.ID)
	actualArea, err := repo.Load(context.Background(), area2.ID)
	actualPath := actualArea.Path
	require.Nil(s.T(), err)
	assert.NotEqual(s.T(), uuid.Nil, area2.Path)
	assert.Equal(s.T(), expectedPath, actualPath) // check that path ( an ltree field ) was populated.
	// *** Create 2nd child area ***
	expectedPath = path.Path{i.ID}
	area3 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area3)
	require.Nil(s.T(), err)
	createdAreaIDs = append(createdAreaIDs, area3.ID)
	actualArea, err = repo.Load(context.Background(), area3.ID)
	require.Nil(s.T(), err)
	actualPath = actualArea.Path
	assert.Equal(s.T(), expectedPath, actualPath)
	// *** Validate that there are 2 children
	childAreaList, err := repo.ListChildren(context.Background(), &i)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(childAreaList))
	for i := 0; i < len(createdAreaIDs); i++ {
		assert.NotNil(s.T(), createdAreaIDs[i], childAreaList[i].ID)
	}
}

func (s *TestAreaRepository) TestListImmediateChildrenOfGrandParents() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestListImmediateChildrenOfGrandParents"
	name2 := "TestListImmediateChildrenOfGrandParents.1"
	name3 := "TestListImmediateChildrenOfGrandParents.1.3"
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: space.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(s.T(), err)
	// *** Create 'son' area ***
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(s.T(), err)
	childAreaList, err := repo.ListChildren(context.Background(), &i)
	assert.Equal(s.T(), 1, len(childAreaList))
	require.Nil(s.T(), err)
	// *** Create 'grandson' area ***
	expectedPath = path.Path{i.ID, area2.ID}
	area4 := area.Area{
		Name:    name3,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area4)
	require.Nil(s.T(), err)
	// when
	childAreaList, err = repo.ListChildren(context.Background(), &i)
	// But , There is only 1 'son' .
	require.Nil(s.T(), err)
	assert.Equal(s.T(), 1, len(childAreaList))
	assert.Equal(s.T(), area2.ID, childAreaList[0].ID)
	// *** Confirm the grandson has no son
	childAreaList, err = repo.ListChildren(context.Background(), &area4)
	assert.Equal(s.T(), 0, len(childAreaList))
}

func (s *TestAreaRepository) TestListParentTree() {
	// given
	repo := area.NewAreaRepository(s.DB)
	name := "TestListParentTree"
	name2 := "TestListParentTree.1"
	newSpace := space.Space{
		Name:    uuid.NewV4().String(),
		OwnerId: s.testIdentity.ID,
	}
	repoSpace := space.NewRepository(s.DB)
	space, err := repoSpace.Create(context.Background(), &newSpace)
	require.Nil(s.T(), err)
	// *** Create Parent Area ***
	i := area.Area{
		Name:    name,
		SpaceID: newSpace.ID,
	}
	err = repo.Create(context.Background(), &i)
	assert.Nil(s.T(), err)
	// *** Create 'son' area ***
	expectedPath := path.Path{i.ID}
	area2 := area.Area{
		Name:    name2,
		SpaceID: space.ID,
		Path:    expectedPath,
	}
	err = repo.Create(context.Background(), &area2)
	require.Nil(s.T(), err)
	listOfCreatedID := []uuid.UUID{i.ID, area2.ID}
	// when
	listOfCreatedAreas, err := repo.LoadMultiple(context.Background(), listOfCreatedID)
	// then
	require.Nil(s.T(), err)
	assert.Equal(s.T(), 2, len(listOfCreatedAreas))
	for i := 0; i < 2; i++ {
		assert.NotNil(s.T(), searchInAreaSlice(listOfCreatedID[i], listOfCreatedAreas))
	}

}
