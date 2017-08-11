package controller_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/area"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/resource"

	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSpaceAreaREST struct {
	gormtestsupport.DBTestSuite
	db             *gormapplication.GormDB
	clean          func()
	svcSpaceAreas  *goa.Service
	ctrlSpaceAreas *SpaceAreasController
}

func TestRunSpaceAreaREST(t *testing.T) {
	resource.Require(t, resource.Database)
	pwd, err := os.Getwd()
	if err != nil {
		require.Nil(t, err)
	}
	suite.Run(t, &TestSpaceAreaREST{DBTestSuite: gormtestsupport.NewDBTestSuite(pwd + "/../config.yaml")})
}

func (rest *TestSpaceAreaREST) SetupTest() {
	rest.db = gormapplication.NewGormDB(rest.DB)
	rest.clean = cleaner.DeleteCreatedEntities(rest.DB)
	rest.svcSpaceAreas, rest.ctrlSpaceAreas = rest.SecuredController()
}

func (rest *TestSpaceAreaREST) TearDownTest() {
	rest.clean()
}

func (rest *TestSpaceAreaREST) SecuredController() (*goa.Service, *SpaceAreasController) {
	pub, _ := wittoken.ParsePublicKey([]byte(wittoken.RSAPublicKey))
	svc := testsupport.ServiceAsUser("Space-Area-Service", wittoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewSpaceAreasController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceAreaREST) SecuredAreasController() (*goa.Service, *AreaController) {
	pub, _ := wittoken.ParsePublicKey([]byte(wittoken.RSAPublicKey))
	svc := testsupport.ServiceAsUser("Area-Service", wittoken.NewManager(pub), testsupport.TestIdentity)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceAreaREST) SecuredAreasControllerWithIdentity(idn *account.Identity) (*goa.Service, *AreaController) {
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Area-Service-With-Identity", wittoken.NewManagerWithPrivateKey(priv), *idn)
	return svc, NewAreaController(svc, rest.db, rest.Configuration)
}

func (rest *TestSpaceAreaREST) UnSecuredController() (*goa.Service, *SpaceAreasController) {
	svc := goa.New("Area-Service")
	return svc, NewSpaceAreasController(svc, rest.db, rest.Configuration)
}

func searchInAreaSlice(searchKey uuid.UUID, areaList *app.AreaList) *app.Area {
	for i := 0; i < len(areaList.Data); i++ {
		if searchKey == *areaList.Data[i].ID {
			return areaList.Data[i]
		}
	}
	return nil
}

func (rest *TestSpaceAreaREST) setupAreas() (area.Area, []uuid.UUID, []area.Area) {
	/*
		Space X --> TestListAreas A---> TestListAreas B
	*/
	var createdAreas []area.Area
	var createdAreaUuids []uuid.UUID
	sp, parentArea := createSpaceAndArea(rest.T(), rest.db)
	createdAreas = append(createdAreas, parentArea)
	createdAreaUuids = append(createdAreaUuids, parentArea.ID)
	parentID := parentArea.ID
	name := "TestListAreas  A"
	ci := newCreateChildAreaPayload(&name)
	owner, err := rest.db.Identities().Load(context.Background(), sp.OwnerId)
	require.Nil(rest.T(), err)
	svc, ctrl := rest.SecuredAreasControllerWithIdentity(owner)
	_, created := test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, parentID.String(), ci)
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.Equal(rest.T(), parentID.String(), *created.Data.Relationships.Parent.Data.ID)
	createdAreaUuids = append(createdAreaUuids, *created.Data.ID)
	createdAreas = append(createdAreas, convertAreaToModel(*created))

	// Create a child of the child created above.
	name = "TestListAreas B"
	ci = newCreateChildAreaPayload(&name)
	newParentID := *created.Data.Relationships.Parent.Data.ID
	_, created = test.CreateChildAreaCreated(rest.T(), svc.Context, svc, ctrl, newParentID, ci)
	assert.Equal(rest.T(), *ci.Data.Attributes.Name, *created.Data.Attributes.Name)
	assert.NotNil(rest.T(), *created.Data.Attributes.CreatedAt)
	assert.NotNil(rest.T(), *created.Data.Attributes.Version)
	assert.Equal(rest.T(), newParentID, *created.Data.Relationships.Parent.Data.ID)
	assert.Contains(rest.T(), *created.Data.Relationships.Children.Links.Self, "children")
	createdAreaUuids = append(createdAreaUuids, *created.Data.ID)
	createdAreas = append(createdAreas, convertAreaToModel(*created))
	return parentArea, createdAreaUuids, createdAreas
}

func assertSpaceAreas(t *testing.T, areaList *app.AreaList, createdAreaUuids []uuid.UUID) {
	assert.Len(t, areaList.Data, 3)
	for i := 0; i < len(createdAreaUuids); i++ {
		assert.NotNil(t, searchInAreaSlice(createdAreaUuids[i], areaList))
	}
}

func (rest *TestSpaceAreaREST) TestListAreasOK() {
	// given
	parentArea, createdAreaUuids, _ := rest.setupAreas()
	// when
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, nil)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasOKUsingExpiredIfModifiedSinceHeader() {
	// given
	parentArea, createdAreaUuids, _ := rest.setupAreas()
	// when
	ifModifiedSince := app.ToHTTPTime(parentArea.UpdatedAt.Add(-1 * time.Hour))
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, &ifModifiedSince, nil)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasOKUsingExpiredIfNoneMatchHeader() {
	// given
	parentArea, createdAreaUuids, _ := rest.setupAreas()
	// when
	ifNoneMatch := "foo"
	res, areaList := test.ListSpaceAreasOK(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, &ifNoneMatch)
	// then
	assertSpaceAreas(rest.T(), areaList, createdAreaUuids)
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasNotModifiedUsingIfModifiedSinceHeader() {
	// given
	parentArea, _, areas := rest.setupAreas()
	// when
	ifModifiedSince := app.ToHTTPTime(areas[len(areas)-1].UpdatedAt)
	res := test.ListSpaceAreasNotModified(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(rest.T(), res)
}

func (rest *TestSpaceAreaREST) TestListAreasNotModifiedUsingIfNoneMatchHeader() {
	// given
	parentArea, _, createdAreas := rest.setupAreas()
	// when
	ifNoneMatch := app.GenerateEntitiesTag([]app.ConditionalRequestEntity{
		createdAreas[0],
		createdAreas[1],
		createdAreas[2],
	})
	res := test.ListSpaceAreasNotModified(rest.T(), rest.svcSpaceAreas.Context, rest.svcSpaceAreas, rest.ctrlSpaceAreas, parentArea.SpaceID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(rest.T(), res)
}
