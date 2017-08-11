package controller_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"

	"context"

	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestAreaREST struct {
	gormtestsupport.DBTestSuite
	db    *gormapplication.GormDB
	clean func()
}

func TestRunAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	require.Nil(t, err)
	suite.Run(t, &TestAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
}

func (rest *TestAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestAreaREST) SecuredController() (*goa.Service, *AreaController) {
	pub, _ := wittoken.ParsePublicKey([]byte(wittoken.RSAPublicKey))
	//priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Area-Service", wittoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) SecuredControllerWithIdentity(idn *account.Identity) (*goa.Service, *AreaController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))

	svc := testsupport.ServiceAsUser("Area-Service", wittoken.NewManagerWithPrivateKey(priv), *idn)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) UnSecuredController() (*goa.Service, *AreaController) {
	svc := goa.New("Area-Service")
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestAreaREST) TestSuccessCreateChildArea() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	parentID := parentArea.ID
	name := "TestSuccessCreateChildArea"
	ci := newCreateChildAreaPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	fmt.Println(*created.Data.Relationships.Parent.Data.ID)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)

	// try creating child area with different identity: should fail
	otherIdentity := &account.Identity{
		Username:     "non-space-owner-identity",
		ProviderType: account.KeycloakIDP,
	}
	errInCreateOther := rest.db.Identities().Create(context.Background(), otherIdentity)
	require.Nil(rest.T(), errInCreateOther)
	svc, ctrl = rest.SecuredControllerWithIdentity(otherIdentity)
	test.CreateChildAreaForbidden(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
}

func (rest *TestAreaREST) TestSuccessCreateMultiChildArea() {
	/*
		TestAreaREST ---> TestSuccessCreateMultiChildArea-0 ----> TestSuccessCreateMultiChildArea-0-0
	*/
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	parentID := parentArea.ID
	name := "TestSuccessCreateMultiChildArea-0"
	ci := newCreateChildAreaPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)
	// Create a child of the child created above.
	name = "TestSuccessCreateMultiChildArea-0-0"
	ci = newCreateChildAreaPayload(&name)
	newParentID := *created.Data.Relationships.Parent.Data.ID
	// when
	_, created = test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, newParentID, ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.NotNil(rest.T(), *created.Data.Attributes.CreatedAt)
	assert.NotNil(rest.T(), *created.Data.Attributes.Version)
	assert.Equal(rest.T(), newParentID, *created.Data.Relationships.Parent.Data.ID)
	assert.Contains(rest.T(), *created.Data.Relationships.Children.Links.Self, "children")
}

func (rest *TestAreaREST) TestConflictCreatDuplicateChildArea() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	parentID := parentArea.ID
	name := uuid.NewV4().String()
	ci := newCreateChildAreaPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	// then
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)

	// try creating the same area again
	test.CreateChildAreaConflict(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)

}

func (rest *TestAreaREST) TestFailCreateChildAreaMissingName() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	parentID := parentArea.ID
	createChildAreaPayload := newCreateChildAreaPayload(nil)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	// when/then
	test.CreateChildAreaBadRequest(rest.T(), svc.Context, svc, ctrl, parentID.String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailCreateChildAreaWithInvalidsParent() {
	// given
	name := "TestFailCreateChildAreaWithInvalidsParent"
	createChildAreaPayload := newCreateChildAreaPayload(&name)
	svc, ctrl := rest.SecuredController()
	// when/then
	test.CreateChildAreaNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailCreateChildAreaNotAuthorized() {
	// given
	_, parentArea := createSpaceAndArea(rest.T(), rest.db)
	parentID := parentArea.ID
	name := "TestFailCreateChildAreaNotAuthorized"
	createChildAreaPayload := newCreateChildAreaPayload(&name)
	svc, ctrl := rest.UnSecuredController()
	// when/then
	test.CreateChildAreaUnauthorized(rest.T(), svc.Context, svc, ctrl, parentID.String(), createChildAreaPayload)
}

func (rest *TestAreaREST) TestFailValidationAreaNameLength() {
	// given
	ci := newCreateChildAreaPayload(&testsupport.TestOversizedNameObj)

	err := ci.Validate()
	// Validate payload function returns an error
	assert.NotNil(rest.T(), err)
	assert.Contains(rest.T(), err.Error(), "length of response.name must be less than or equal to than 62")
}

func (rest *TestAreaREST) TestFailValidationAreaNameStartWith() {
	// given
	name := "_TestSuccessCreateChildArea"
	ci := newCreateChildAreaPayload(&name)

	err := ci.Validate()
	// Validate payload function returns an error
	assert.NotNil(rest.T(), err)
	assert.Contains(rest.T(), err.Error(), "response.name must match the regexp")
}

func (rest *TestAreaREST) TestFailShowAreaNotFound() {
	// given
	svc, ctrl := rest.SecuredController()
	// when/then
	test.ShowAreaNotFound(rest.T(), svc.Context, svc, ctrl, uuid.NewV4().String(), nil, nil)
}

func (rest *TestAreaREST) TestShowAreaOK() {
	// given
	_, a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaOKUsingExpiredIfModifedSinceHeader() {
	// given
	_, a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSince := app.ToHTTPTime(a.UpdatedAt.Add(-1 * time.Hour))
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaOKUsingExpiredIfNoneMatchHeader() {
	// given
	_, a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := "foo"
	res, _ := test.ShowAreaOK(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaNotModifiedUsingIfModifedSinceHeader() {
	// given
	_, a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifModifiedSince := app.ToHTTPTime(a.UpdatedAt)
	res := test.ShowAreaNotModified(rest.T(), svc.Context, svc, ctrl, a.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowAreaNotModifiedIfNoneMatchHeader() {
	// given
	_, a := createSpaceAndArea(rest.T(), rest.db)
	svc, ctrl := rest.SecuredController()
	// when
	ifNoneMatch := app.GenerateEntityTag(a)
	res := test.ShowAreaNotModified(rest.T(), svc.Context, svc, ctrl, a.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) createChildArea(name string, parent area.Area, svc *goa.Service, ctrl *AreaController) *app.AreaSingle {
	ci := newCreateChildAreaPayload(&name)
	// when
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parent.ID.String(), ci)
	return created
}

func (rest *TestAreaREST) TestShowChildrenAreaOK() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	rest.createChildArea("TestShowChildrenAreaOK", parentArea, svc, ctrl)
	// when
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, nil)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaOKUsingExpiredIfModifedSinceHeader() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	rest.createChildArea("TestShowChildrenAreaOKUsingExpiredIfModifedSinceHeader", parentArea, svc, ctrl)
	// when
	ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaOKUsingExpiredIfNoneMatchHeader() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	rest.createChildArea("TestShowChildrenAreaOKUsingExpiredIfNoneMatchHeader", parentArea, svc, ctrl)
	// when
	ifNoneMatch := "foo"
	res, result := test.ShowChildrenAreaOK(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
	//then
	assert.Equal(rest.T(), 1, len(result.Data))
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaNotModifiedUsingIfModifedSinceHeader() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	childArea := rest.createChildArea("TestShowChildrenAreaNotModifiedUsingIfModifedSinceHeader", parentArea, svc, ctrl)
	// when
	ifModifiedSince := app.ToHTTPTime(*childArea.Data.Attributes.UpdatedAt)
	res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), &ifModifiedSince, nil)
	//then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestAreaREST) TestShowChildrenAreaNotModifiedIfNoneMatchHeader() {
	// given
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredControllerWithIdentity(owner)
	childArea := rest.createChildArea("TestShowChildrenAreaNotModifiedIfNoneMatchHeader", parentArea, svc, ctrl)
	modelChildArea := convertAreaToModel(*childArea)
	// when
	ifNoneMatch := app.GenerateEntityTag(modelChildArea)
	res := test.ShowChildrenAreaNotModified(rest.T(), svc.Context, svc, ctrl, parentArea.ID.String(), nil, &ifNoneMatch)
	//then
	assertResponseHeaders(rest.T(), res)
}

func convertAreaToModel(appArea app.AreaSingle) area.Area {
	return area.Area{
		ID:      *appArea.Data.ID,
		Version: *appArea.Data.Attributes.Version,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *appArea.Data.Attributes.UpdatedAt,
		},
	}
}

func newCreateChildAreaPayload(name *string) *app.CreateChildAreaPayload {
	areaType := area.APIStringTypeAreas
	return &app.CreateChildAreaPayload{
		Data: &app.Area{
			Type: areaType,
			Attributes: &app.AreaAttributes{
				Name: name,
			},
		},
	}
}

func createSpaceAndArea(t *testing.T, db *gormapplication.GormDB) (space.Space, area.Area) {
	var areaObj area.Area
	var spaceObj space.Space
	application.Transactional(db, func(app application.Application) error {
		owner := &account.Identity{
			Username:     "new-space-owner-identity",
			ProviderType: account.KeycloakIDP,
		}
		errCreateOwner := app.Identities().Create(context.Background(), owner)
		require.Nil(t, errCreateOwner)

		spaceObj = space.Space{
			Name:    "TestAreaREST-" + uuid.NewV4().String(),
			OwnerId: owner.ID,
		}
		_, err := app.Spaces().Create(context.Background(), &spaceObj)
		require.Nil(t, err)
		name := "Main Area-" + uuid.NewV4().String()
		areaObj = area.Area{
			Name:    name,
			SpaceID: spaceObj.ID,
		}
		err = app.Areas().Create(context.Background(), &areaObj)
		require.Nil(t, err)
		return nil
	})
	log.Info(nil, nil, "Space and root area created")
	return spaceObj, areaObj
}
