package space_test

import (
	"fmt"
	"testing"

	"context"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var testSpace string = uuid.NewV4().String()
var testSpace2 string = uuid.NewV4().String()

func TestRunRepoBBTest(t *testing.T) {
	suite.Run(t, &repoBBTest{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

type repoBBTest struct {
	gormtestsupport.DBTestSuite
	repo         space.Repository
	testIdentity account.Identity
	clean        func()
}

func (test *repoBBTest) SetupTest() {
	test.repo = space.NewRepository(test.DB)
	test.clean = cleaner.DeleteCreatedEntities(test.DB)
	testIdentity, err := testsupport.CreateTestIdentity(test.DB, "WorkItemSuite setup user", "test provider")
	require.Nil(test.T(), err)
	test.testIdentity = *testIdentity
}

func (test *repoBBTest) TearDownTest() {
	test.clean()
}

func (test *repoBBTest) TestCreate() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)
	require.Equal(test.T(), res.Name, testSpace)
}

func (test *repoBBTest) TestCreateFailSpaceNameChecks() {
	expectSpace(test.create(""), test.assertBadParameter())
}

func (test *repoBBTest) TestCreateFailSameOwner() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	require.Equal(test.T(), res.Name, testSpace)

	expectSpace(test.create(testSpace), test.assertDataConflict())
}

func (test *repoBBTest) TestLoad() {
	expectSpace(test.load(uuid.NewV4()), test.assertNotFound())
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	res2, _ := expectSpace(test.load(res.ID), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *repoBBTest) TestExistsSpace() {
	t := test.T()
	resource.Require(t, resource.Database)

	t.Run("space exists", func(t *testing.T) {
		// given
		err := test.repo.CheckExists(context.Background(), space.SystemSpace.String())
		require.Nil(t, err)
	})

	t.Run("space doesn't exist", func(t *testing.T) {
		err := test.repo.CheckExists(context.Background(), uuid.NewV4().String())

		require.IsType(t, errors.NotFoundError{}, err)
	})

}

func (test *repoBBTest) TestSaveOk() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	newName := uuid.NewV4().String()
	res.Name = newName
	res2, _ := expectSpace(test.save(*res), test.requireOk)
	assert.Equal(test.T(), newName, res2.Name)
}

func (test *repoBBTest) TestSaveFail() {
	p1, _ := expectSpace(test.create(testSpace), test.requireOk)
	p2, _ := expectSpace(test.create(testSpace2), test.requireOk)

	p1.Name = ""
	expectSpace(test.save(*p1), test.assertBadParameter())

	p1.Name = p2.Name
	expectSpace(test.save(*p1), test.assertBadParameter())
}

func (test *repoBBTest) TestSaveNew() {
	p := space.Space{
		ID:      uuid.NewV4(),
		Version: 0,
		Name:    testSpace,
	}

	expectSpace(test.save(p), test.requireErrorType(errors.NotFoundError{}))
}

func (test *repoBBTest) TestDelete() {
	res, _ := expectSpace(test.create(testSpace), test.requireOk)
	expectSpace(test.load(res.ID), test.requireOk)
	expectSpace(test.delete(res.ID), func(p *space.Space, err error) { require.Nil(test.T(), err) })
	expectSpace(test.load(res.ID), test.assertNotFound())
	expectSpace(test.delete(uuid.NewV4()), test.assertNotFound())
	expectSpace(test.delete(uuid.Nil), test.assertNotFound())
}

func (test *repoBBTest) TestList() {
	// given
	_, orgCount, _ := test.list(nil, nil)

	newSpace, err := expectSpace(test.create(testSpace), test.requireOk)

	require.Nil(test.T(), err)
	require.NotNil(test.T(), newSpace)

	// when
	updatedListOfSpaces, newCount, _ := test.list(nil, nil)

	test.T().Log(fmt.Sprintf("Old count of spaces : %d , new count of spaces : %d", orgCount, newCount))

	foundNewSpaceInList := false
	for _, retrievedSpace := range updatedListOfSpaces {
		if retrievedSpace.ID == newSpace.ID {
			foundNewSpaceInList = true
		}
	}
	// then
	assert.True(test.T(), foundNewSpaceInList)
}

func (test *repoBBTest) TestListDoNotReturnPointerToSameObject() {
	expectSpace(test.create(testSpace), test.requireOk)
	expectSpace(test.create(testSpace2), test.requireOk)
	spaces, newCount, _ := test.list(nil, nil)
	assert.True(test.T(), newCount >= 2)
	assert.True(test.T(), spaces[0].Name != spaces[1].Name)
}

func (test *repoBBTest) TestLoadSpaceByName() {
	expectSpace(test.load(uuid.NewV4()), test.assertNotFound())
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	res2, _ := expectSpace(test.loadByUserIdAndName(test.testIdentity.ID, res.Name), test.requireOk)
	assert.True(test.T(), (*res).Equal(*res2))
}

func (test *repoBBTest) TestLoadSpaceByNameDifferentOwner() {
	expectSpace(test.load(uuid.NewV4()), test.assertNotFound())
	res, _ := expectSpace(test.create(testSpace), test.requireOk)

	_, err := expectSpace(test.loadByUserIdAndName(uuid.NewV4(), res.Name), test.requireErrorType(errors.NotFoundError{}))
	assert.NotNil(test.T(), err)
}

func (test *repoBBTest) TestLoadSpaceByNameNonExistentSpaceName() {
	expectSpace(test.load(uuid.NewV4()), test.assertNotFound())
	expectSpace(test.create(testSpace), test.requireOk)

	_, err := expectSpace(test.loadByUserIdAndName(test.testIdentity.ID, uuid.NewV4().String()), test.requireErrorType(errors.NotFoundError{}))
	assert.NotNil(test.T(), err)
}

type spaceExpectation func(p *space.Space, err error)

func expectSpace(f func() (*space.Space, error), e spaceExpectation) (*space.Space, error) {
	p, err := f()
	e(p, err)
	return p, errs.WithStack(err)
}

func (test *repoBBTest) requireOk(p *space.Space, err error) {
	assert.NotNil(test.T(), p)
	require.Nil(test.T(), err)
}

func (test *repoBBTest) assertNotFound() func(p *space.Space, err error) {
	return test.assertErrorType(errors.NotFoundError{})
}
func (test *repoBBTest) assertBadParameter() func(p *space.Space, err error) {
	return test.assertErrorType(errors.BadParameterError{})
}

func (test *repoBBTest) assertDataConflict() func(p *space.Space, err error) {
	return test.assertErrorType(errors.DataConflictError{})
}

func (test *repoBBTest) assertErrorType(e error) func(p *space.Space, e2 error) {
	return func(p *space.Space, err error) {
		assert.Nil(test.T(), p)
		assert.IsType(test.T(), e, err, "error was %v", err)
	}
}

func (test *repoBBTest) requireErrorType(e error) func(p *space.Space, err error) {
	return func(p *space.Space, err error) {
		assert.Nil(test.T(), p)
		require.IsType(test.T(), e, err)
	}
}

func (test *repoBBTest) create(name string) func() (*space.Space, error) {
	newSpace := space.Space{
		Name:    name,
		OwnerId: test.testIdentity.ID,
	}
	return func() (*space.Space, error) { return test.repo.Create(context.Background(), &newSpace) }
}

func (test *repoBBTest) save(p space.Space) func() (*space.Space, error) {
	return func() (*space.Space, error) { return test.repo.Save(context.Background(), &p) }
}

func (test *repoBBTest) load(id uuid.UUID) func() (*space.Space, error) {
	return func() (*space.Space, error) { return test.repo.Load(context.Background(), id) }
}

func (test *repoBBTest) loadByUserIdAndName(userId uuid.UUID, spaceName string) func() (*space.Space, error) {
	return func() (*space.Space, error) {
		return test.repo.LoadByOwnerAndName(context.Background(), &userId, &spaceName)
	}
}

func (test *repoBBTest) delete(id uuid.UUID) func() (*space.Space, error) {
	return func() (*space.Space, error) { return nil, test.repo.Delete(context.Background(), id) }
}

func (test *repoBBTest) list(start *int, length *int) ([]space.Space, uint64, error) {
	return test.repo.List(context.Background(), start, length)
}
