package link_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type linkRepoBlackBoxTest struct {
	gormtestsupport.DBTestSuite
	workitemLinkRepo         *link.GormWorkItemLinkRepository
	workitemLinkTypeRepo     link.WorkItemLinkTypeRepository
	workitemLinkCategoryRepo link.WorkItemLinkCategoryRepository
	workitemRepo             workitem.WorkItemRepository
	clean                    func()
	ctx                      context.Context
	testSpace                uuid.UUID
	testIdentity             account.Identity
	linkCategoryID           uuid.UUID
	testTreeLinkTypeID       uuid.UUID
	parent1                  *workitem.WorkItem
	parent2                  *workitem.WorkItem
	child                    *workitem.WorkItem
}

// SetupSuite overrides the DBTestSuite's function but calls it before doing anything else
// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *linkRepoBlackBoxTest) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.ctx = migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(s.ctx)
}

func TestRunLinkRepoBlackBoxTest(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &linkRepoBlackBoxTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *linkRepoBlackBoxTest) SetupTest() {
	s.workitemRepo = workitem.NewWorkItemRepository(s.DB)
	s.workitemLinkRepo = link.NewWorkItemLinkRepository(s.DB)
	s.workitemLinkTypeRepo = link.NewWorkItemLinkTypeRepository(s.DB)
	s.workitemLinkCategoryRepo = link.NewWorkItemLinkCategoryRepository(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "jdoe1", "test")
	s.testIdentity = *testIdentity
	require.Nil(s.T(), err)

	// create a space
	spaceRepository := space.NewRepository(s.DB)
	spaceName := testsupport.CreateRandomValidTestName("test-space")
	testSpace, err := spaceRepository.Create(s.ctx, &space.Space{
		Name:    spaceName,
		OwnerId: testIdentity.ID,
	})
	require.Nil(s.T(), err)
	s.testSpace = testSpace.ID

	// Create a work item link category
	categoryName := "test" + uuid.NewV4().String()
	categoryDescription := "Test Link Category"
	linkCategoryModel1 := link.WorkItemLinkCategory{
		Name:        categoryName,
		Description: &categoryDescription,
	}
	linkCategory, err := s.workitemLinkCategoryRepo.Create(s.ctx, &linkCategoryModel1)
	require.Nil(s.T(), err)
	s.linkCategoryID = linkCategory.ID

	// create tree topology link type
	treeLinkTypeModel := link.WorkItemLinkType{
		Name:           "Parent child item",
		ForwardName:    "parent of",
		ReverseName:    "child of",
		Topology:       "tree",
		LinkCategoryID: linkCategory.ID,
		SpaceID:        s.testSpace,
	}
	testTreeLinkType, err := s.workitemLinkTypeRepo.Create(s.ctx, &treeLinkTypeModel)
	require.Nil(s.T(), err)
	s.testTreeLinkTypeID = testTreeLinkType.ID
	// create 3 workitems for linking (or not) during the tests
	s.parent1, err = s.createWorkitem(workitem.SystemBug, "Parent 1", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	s.parent2, err = s.createWorkitem(workitem.SystemBug, "Parent 2", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	s.child, err = s.createWorkitem(workitem.SystemBug, "Child", workitem.SystemStateNew)
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TearDownTest() {
	s.clean()
}

func (s *linkRepoBlackBoxTest) createWorkitem(wiType uuid.UUID, title, state string) (*workitem.WorkItem, error) {
	return s.workitemRepo.Create(
		s.ctx, s.testSpace, wiType,
		map[string]interface{}{
			workitem.SystemTitle: title,
			workitem.SystemState: state,
		}, s.testIdentity.ID)
}

// This creates a parent-child link between two workitems -> parent1 and Child. It tests that when there is an attempt to create another parent (parent2) of child, it should throw an error.
func (s *linkRepoBlackBoxTest) TestDisallowMultipleParents() {
	// create a work item link
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent2.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then
	require.NotNil(s.T(), err)
}

// TestCountChildWorkitems tests total number of workitem children returned by list is equal to the total number of workitem children created
// and total number of workitem children in a page are equal to the "limit" specified
func (s *linkRepoBlackBoxTest) TestCountChildWorkitems() {
	// create 3 workitems for linking as children to parent workitem
	child1, err := s.createWorkitem(workitem.SystemBug, "Child 1", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	child2, err := s.createWorkitem(workitem.SystemBug, "Child 2", workitem.SystemStateNew)
	require.Nil(s.T(), err)
	child3, err := s.createWorkitem(workitem.SystemBug, "Child 3", workitem.SystemStateNew)
	require.Nil(s.T(), err)

	// link the children workitems to parent
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, child1.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, child2.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, child3.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	offset := 0
	limit := 1
	res, count, err := s.workitemLinkRepo.ListWorkItemChildren(s.ctx, s.parent1.ID, &offset, &limit)
	require.Nil(s.T(), err)
	require.Len(s.T(), res, 1)
	require.Equal(s.T(), 3, int(count))
}

func (s *linkRepoBlackBoxTest) TestWorkItemHasNoChildAfterDeletion() {
	// given
	// create a work item link...
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wil, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// ... and remove it
	err = s.workitemLinkRepo.Delete(s.ctx, wil.ID, s.testIdentity.ID)
	require.Nil(s.T(), err)

	// when
	hasChildren, err := s.workitemLinkRepo.WorkItemHasChildren(s.ctx, s.parent1.ID)
	// then
	assert.Nil(s.T(), err)
	assert.False(s.T(), hasChildren)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkNoLink() {
	// given link type exists but no link to child item
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.child.ID, *linkType)
	// then: there must be no error because no link exists
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkLinkExistsButIgnored() {
	// given
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, &s.parent1.ID, s.child.ID, *linkType)
	// then: there must be no error because the existing link was ignored
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyOkNoLinkWithSameType() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// use another link type to validate
	linkTypeModel := link.WorkItemLinkType{
		Name:           "foo/bar relationship",
		ForwardName:    "foo",
		ReverseName:    "bar",
		Topology:       "tree",
		LinkCategoryID: s.linkCategoryID,
		SpaceID:        s.testSpace,
	}
	foobarLinkType, err := s.workitemLinkTypeRepo.Create(s.ctx, &linkTypeModel)
	require.Nil(s.T(), err)
	// when
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.child.ID, *foobarLinkType)
	// then: there must be no error because no link of the same type exists
	assert.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyErrorLinkExists() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when checking the child *without* excluding the parent item
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, nil, s.child.ID, *linkType)
	// then: there must be an error because a link of the same type already exists
	assert.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestValidateTopologyErrorAnotherLinkExists() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	linkType, err := s.workitemLinkTypeRepo.Load(s.ctx, s.testTreeLinkTypeID)
	require.Nil(s.T(), err)
	// when checking the child  while excluding the parent item
	err = s.workitemLinkRepo.ValidateTopology(s.ctx, &s.parent2.ID, s.child.ID, *linkType)
	// then: there must be an error because a link of the same type already exists with another parent
	assert.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestCreateLinkOK() {
	// given
	// when
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestUpdateLinkOK() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	wiLink, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when
	s.T().Log(fmt.Sprintf("updating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err = s.workitemLinkRepo.Save(s.ctx, *wiLink, s.testIdentity.ID)
	// then
	require.Nil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestCreateLinkErrorOtherParentChildLinkExist() {
	// given
	// link 2 workitems together
	s.T().Log(fmt.Sprintf("creating link with treelinktype.ID=%v", s.testTreeLinkTypeID))
	_, err := s.workitemLinkRepo.Create(s.ctx, s.parent1.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	require.Nil(s.T(), err)
	// when try to link parent#2 to child
	_, err = s.workitemLinkRepo.Create(s.ctx, s.parent2.ID, s.child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
	// then expect an error because a parent/link relation already exists with the child item
	require.NotNil(s.T(), err)
}

func (s *linkRepoBlackBoxTest) TestExistsLink() {
	t := s.T()
	resource.Require(t, resource.Database)

	t.Run("link exists", func(t *testing.T) {
		// given
		// create 3 workitems for linking
		workitemRepository := workitem.NewWorkItemRepository(s.DB)
		Parent1, err := workitemRepository.Create(
			s.ctx, s.testSpace, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle: "Parent 1",
				workitem.SystemState: workitem.SystemStateNew,
			}, s.testIdentity.ID)
		require.Nil(s.T(), err)

		Child, err := workitemRepository.Create(
			s.ctx, s.testSpace, workitem.SystemBug,
			map[string]interface{}{
				workitem.SystemTitle: "Child",
				workitem.SystemState: workitem.SystemStateNew,
			}, s.testIdentity.ID)
		require.Nil(s.T(), err)

		// Create a work item link category
		linkCategoryRepository := link.NewWorkItemLinkCategoryRepository(s.DB)
		categoryName := "test exists" + uuid.NewV4().String()
		categoryDescription := "Test Exists Link Category"
		linkCategoryModel1 := link.WorkItemLinkCategory{
			Name:        categoryName,
			Description: &categoryDescription,
		}
		linkCategory, err := linkCategoryRepository.Create(s.ctx, &linkCategoryModel1)
		require.Nil(s.T(), err)

		// create tree topology link type
		linkTypeRepository := link.NewWorkItemLinkTypeRepository(s.DB)
		linkTypeModel1 := link.WorkItemLinkType{
			Name:           "TestExistsLinkType",
			ForwardName:    "foo",
			ReverseName:    "foo",
			Topology:       "tree",
			LinkCategoryID: linkCategory.ID,
			SpaceID:        s.testSpace,
		}
		TestTreeLinkType, err := linkTypeRepository.Create(s.ctx, &linkTypeModel1)
		require.Nil(s.T(), err)
		s.testTreeLinkTypeID = TestTreeLinkType.ID

		// create a work item link
		linkRepository := link.NewWorkItemLinkRepository(s.DB)
		linkTest, err := linkRepository.Create(s.ctx, Parent1.ID, Child.ID, s.testTreeLinkTypeID, s.testIdentity.ID)
		require.Nil(s.T(), err)

		err = linkRepository.CheckExists(s.ctx, linkTest.ID.String())
		require.Nil(s.T(), err)
	})

	t.Run("link doesn't exist", func(t *testing.T) {
		// then
		linkRepository := link.NewWorkItemLinkRepository(s.DB)
		// when
		err := linkRepository.CheckExists(s.ctx, uuid.NewV4().String())
		// then

		require.IsType(t, errors.NotFoundError{}, err)
	})

}
