package controller_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/application"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	. "github.com/fabric8-services/fabric8-wit/controller"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSuiteWorkItemLinkCategory(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(workItemLinkCategorySuite))
}

//-----------------------------------------------------------------------------
// Test Suite setup
//-----------------------------------------------------------------------------

// The workItemLinkCategorySuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type workItemLinkCategorySuite struct {
	suite.Suite
	db          *gorm.DB
	appDB       application.DB
	linkCatCtrl *WorkItemLinkCategoryController
	svc         *goa.Service
}

var wilCatConfiguration *config.ConfigurationData

func init() {
	var err error
	wilCatConfiguration, err = config.GetConfigurationData()
	if err != nil {
		panic(fmt.Errorf("Failed to setup the configuration: %s", err.Error()))
	}
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *workItemLinkCategorySuite) SetupSuite() {
	var err error
	s.db, err = gorm.Open("postgres", wilCatConfiguration.GetPostgresConfigString())
	require.Nil(s.T(), err)
	s.appDB = gormapplication.NewGormDB(s.db)
	priv, _ := wittoken.ParsePrivateKey([]byte(wittoken.RSAPrivateKey))
	s.svc = testsupport.ServiceAsUser("workItemLinkSpace-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	require.NotNil(s.T(), s.svc)
	s.linkCatCtrl = NewWorkItemLinkCategoryController(s.svc, gormapplication.NewGormDB(s.db))
	require.NotNil(s.T(), s.linkCatCtrl)
}

// The TearDownSuite method will run after all the tests in the suite have been run
// It tears down the database connection for all the tests in this suite.
func (s *workItemLinkCategorySuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// removeWorkItemLinkCategories removes all work item link categories from the db that will be created
// during these tests. We need to remove them completely and not only set the
// "deleted_at" field, which is why we need the Unscoped() function.
func (s *workItemLinkCategorySuite) removeWorkItemLinkCategories() {
	s.db.Unscoped().Delete(&link.WorkItemLinkCategory{Name: "test-system"})
	s.db.Unscoped().Delete(&link.WorkItemLinkCategory{Name: "test-user"})
}

// The SetupTest method will be run before every test in the suite.
// SetupTest ensures that none of the work item link categories that we will create already exist.
func (s *workItemLinkCategorySuite) SetupTest() {
	s.removeWorkItemLinkCategories()
}

// The TearDownTest method will be run after every test in the suite.
func (s *workItemLinkCategorySuite) TearDownTest() {
	s.removeWorkItemLinkCategories()
}

//-----------------------------------------------------------------------------
// helper method
//-----------------------------------------------------------------------------

// createWorkItemLinkCategorySystem defines a work item link category "test-system"
func (s *workItemLinkCategorySuite) createWorkItemLinkCategorySystem() (http.ResponseWriter, *app.WorkItemLinkCategorySingle) {
	name := "test-system"
	description := "This work item link category is reserved for the core system."
	id := uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231")

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}

	return test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, &payload)
}

// createWorkItemLinkCategoryUser defines a work item link category "test-user"
func (s *workItemLinkCategorySuite) createWorkItemLinkCategoryUser() (http.ResponseWriter, *app.WorkItemLinkCategorySingle) {
	name := "test-user"
	description := "This work item link category is managed by an admin user."
	id := uuid.FromStringOrNil("bf30167a-9446-42de-82be-6b3815152051")

	// Use the goa generated code to create a work item link category
	payload := app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}

	return test.CreateWorkItemLinkCategoryCreated(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, &payload)
}

func createWorkItemLinkCategorySystemInRepo(t *testing.T, db application.DB, ctx context.Context) uuid.UUID {
	description := "This work item link category is reserved for the core system."
	linkCat := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "test-system",
		Description: &description,
	}
	return createWorkItemLinkCategoryInRepo(t, db, ctx, linkCat)
}

func createWorkItemLinkCategoryUserInRepo(t *testing.T, db application.DB, ctx context.Context) uuid.UUID {
	description := "This work item link category is managed by an admin user."
	linkCat := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("bf30167a-9446-42de-82be-6b3815152051"),
		Name:        "test-user",
		Description: &description,
	}
	return createWorkItemLinkCategoryInRepo(t, db, ctx, linkCat)
}

func createWorkItemLinkCategoryInRepo(t *testing.T, db application.DB, ctx context.Context, linkCat link.WorkItemLinkCategory) uuid.UUID {
	err := application.Transactional(db, func(appl application.Application) error {
		_, err := appl.WorkItemLinkCategories().Create(ctx, &linkCat)
		return err
	})
	require.Nil(t, err)
	return linkCat.ID
}

//-----------------------------------------------------------------------------
// Actual tests
//-----------------------------------------------------------------------------

func (s *workItemLinkCategorySuite) TestCreateAndDeleteWorkItemLinkCategoryFails() {
	description := "This work item link category is managed by an admin user."

	appLinkCat := ConvertLinkCategoryFromModel(link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("bf30167a-9446-42de-82be-6b3815152051"),
		Name:        "test-user",
		Description: &description,
	})
	payload := app.CreateWorkItemLinkCategoryPayload{Data: appLinkCat.Data}

	test.CreateWorkItemLinkCategoryMethodNotAllowed(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, &payload)
	test.DeleteWorkItemLinkCategoryMethodNotAllowed(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *payload.Data.ID)
}

func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryFails() {
	description := "New description for work item link category."

	appLinkCat := ConvertLinkCategoryFromModel(link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19"),
		Name:        "Some name",
		Description: &description,
	})
	payload := app.UpdateWorkItemLinkCategoryPayload{Data: appLinkCat.Data}

	test.UpdateWorkItemLinkCategoryMethodNotAllowed(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *payload.Data.ID, &payload)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
// TestCreateWorkItemLinkCategory tests if we can create the "test-system" work item link category
func (s *workItemLinkCategorySuite) TestCreateAndDeleteWorkItemLinkCategory() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)

	_, linkCatUser := s.createWorkItemLinkCategoryUser()
	require.NotNil(s.T(), linkCatUser)

	test.DeleteWorkItemLinkCategoryOK(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *linkCatSystem.Data.ID)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestCreateWorkItemLinkCategoryBadRequest() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	description := "New description for work item link category."
	name := "" // This will lead to a bad parameter error
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cdBB")
	payload := &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	err := payload.Validate()

	// Validate payload function returns an error
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "response.name must match the regexp")
}

func (s *workItemLinkCategorySuite) TestFailValidationWorkItemLinkCategoryNameLength() {
	// given
	description := "New description for work item link category."
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cdBB")
	payload := &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &testsupport.TestOversizedNameObj,
				Description: &description,
			},
		},
	}

	err := payload.Validate()

	// Validate payload function returns an error
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "length of response.name must be less than or equal to than 62")
}

func (s *workItemLinkCategorySuite) TestFailValidationWorkItemLinkCategoryNameStartWith() {
	// given
	description := "New description for work item link category."
	name := "_Name" // This will lead to a bad parameter error
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cdBB")
	payload := &app.CreateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}

	err := payload.Validate()
	// Validate payload function returns an error
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "response.name must match the regexp")
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestDeleteWorkItemLinkCategoryNotFound() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	test.DeleteWorkItemLinkCategoryNotFound(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, uuid.FromStringOrNil("01f6c751-53f3-401f-be9b-6a9a230db8AA"))
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryNotFound() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	name := "Some name"
	description := "New description for work item link category."
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19")
	payload := &app.UpdateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &name,
				Description: &description,
			},
		},
	}
	test.UpdateWorkItemLinkCategoryNotFound(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *payload.Data.ID, payload)
}

// func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryBadRequestDueToBadID() {
// 	description := "New description for work item link category."
// 	id := "something that is not a UUID" // This will cause a Not Found error
// 	payload := &app.UpdateWorkItemLinkCategoryPayload{
// 		Data: &app.WorkItemLinkCategoryData{
// 			ID:   &id,
// 			Type: workitem.EndpointWorkItemLinkCategories,
// 			Attributes: &app.WorkItemLinkCategoryAttributes{
// 				Description: &description,
// 			},
// 		},
// 	}
// 	test.UpdateWorkItemLinkCategoryBadRequest(s.T(), nil, nil, s.linkCatCtrl, *payload.Data.ID, payload)
// }

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) xUpdateWorkItemLinkCategoryBadRequestDueToBadType() {
	description := "New description for work item link category."
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19")
	payload := &app.UpdateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: "something that is not workitemlinkcategories", // this will cause a BadParameter error
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Description: &description,
			},
		},
	}
	test.UpdateWorkItemLinkCategoryBadRequest(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *payload.Data.ID, payload)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) xUpdateWorkItemLinkCategoryBadRequestDueToEmptyName() {
	name := "" // When updating the name, it must not be empty
	id := uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19")
	payload := &app.UpdateWorkItemLinkCategoryPayload{
		Data: &app.WorkItemLinkCategoryData{
			ID:   &id,
			Type: link.EndpointWorkItemLinkCategories,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name: &name,
			},
		},
	}
	test.UpdateWorkItemLinkCategoryBadRequest(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *payload.Data.ID, payload)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryBadRequestDueToVersionConflictError() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)
	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{
		Data: linkCatSystem.Data,
	}
	newVersion := *linkCatSystem.Data.Attributes.Version + 42 // This will cause a version conflict error
	updatePayload.Data.Attributes.Version = &newVersion
	test.UpdateWorkItemLinkCategoryConflict(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryOK() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	// given
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)
	description := "New description for work item link category \"system\"."
	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{}
	updatePayload.Data = linkCatSystem.Data
	updatePayload.Data.Attributes.Description = &description
	// when
	_, newLinkCat := test.UpdateWorkItemLinkCategoryOK(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)
	// then
	// Test that description was updated and version got incremented
	require.NotNil(s.T(), newLinkCat.Data.Attributes.Description)
	assert.Equal(s.T(), description, *newLinkCat.Data.Attributes.Description)
	require.NotNil(s.T(), newLinkCat.Data.Attributes.Version)
	assert.Equal(s.T(), *linkCatSystem.Data.Attributes.Version+1, *newLinkCat.Data.Attributes.Version)
}

// Currently not used. Disabled as part of https://github.com/fabric8-services/fabric8-wit/issues/1299
func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryConflict() {
	s.T().Skip("skipped because Work Item Link Category Create/Update/Delete endpoints are disabled")
	// given
	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
	require.NotNil(s.T(), linkCatSystem)
	description := "New description for work item link category \"system\"."
	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{}
	updatePayload.Data = linkCatSystem.Data
	updatePayload.Data.Attributes.Description = &description
	version := 123456
	updatePayload.Data.Attributes.Version = &version
	// when/then
	test.UpdateWorkItemLinkCategoryConflict(s.T(), s.svc.Context, s.svc, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)
}

//func (s *workItemLinkCategorySuite) TestUpdateWorkItemLinkCategoryBadRequest() {
//	_, linkCatSystem := s.createWorkItemLinkCategorySystem()
//	require.NotNil(s.T(), linkCatSystem)
//
//	description := "New description for work item link category \"system\"."
//	updatePayload := &app.UpdateWorkItemLinkCategoryPayload{}
//	updatePayload.Data = linkCatSystem.Data
//	updatePayload.Data.Attributes.Description = &description
//	updatePayload.Data.Type = "this is a wrong type!!!" // "should be workitemlinkcategories"
//
//	test.UpdateWorkItemLinkCategoryBadRequest(s.T(), nil, nil, s.linkCatCtrl, *linkCatSystem.Data.ID, updatePayload)
//}

// TestShowWorkItemLinkCategoryOK tests if we can fetch the "test-system" work item link category
func (s *workItemLinkCategorySuite) TestShowWorkItemLinkCategoryOK() {
	// Create the work item link category first and try to read it back in
	id := createWorkItemLinkCategorySystemInRepo(s.T(), s.appDB, s.linkCatCtrl.Context)
	_, linkCat2 := test.ShowWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl, id)

	require.NotNil(s.T(), linkCat2)
	require.NotNil(s.T(), linkCat2.Data.Links, "The link category MUST include a self link")
	require.NotEmpty(s.T(), linkCat2.Data.Links.Self, "The link category MUST include a self link that's not empty")
	require.Len(s.T(), linkCat2.Included, 0, "The link category has nothing to include")
}

// TestShowWorkItemLinkCategoryNotFound tests if we can fetch a non existing work item link category
func (s *workItemLinkCategorySuite) TestShowWorkItemLinkCategoryNotFound() {
	test.ShowWorkItemLinkCategoryNotFound(s.T(), nil, nil, s.linkCatCtrl, uuid.FromStringOrNil("88727441-4a21-4b35-aabe-007f8273cd19"))
}

// TestListWorkItemLinkCategoryOK tests if we can find the work item link categories
// "test-system" and "test-user" in the list of work item link categories
func (s *workItemLinkCategorySuite) TestListWorkItemLinkCategoryOK() {
	createWorkItemLinkCategorySystemInRepo(s.T(), s.appDB, s.linkCatCtrl.Context)
	createWorkItemLinkCategoryUserInRepo(s.T(), s.appDB, s.linkCatCtrl.Context)

	// Fetch a single work item link category
	_, linkCatCollection := test.ListWorkItemLinkCategoryOK(s.T(), nil, nil, s.linkCatCtrl)

	require.NotNil(s.T(), linkCatCollection)
	require.Nil(s.T(), linkCatCollection.Validate())

	// Check the number of found work item link categories
	require.NotNil(s.T(), linkCatCollection.Data)
	require.Condition(s.T(), func() bool {
		return (len(linkCatCollection.Data) >= 2)
	}, "At least two work item link categories must exist (system and user), but only %d exist.", len(linkCatCollection.Data))

	// Search for the work item types that must exist at minimum
	toBeFound := 2
	for i := 0; i < len(linkCatCollection.Data) && toBeFound > 0; i++ {
		if *linkCatCollection.Data[i].Attributes.Name == "test-system" || *linkCatCollection.Data[i].Attributes.Name == "test-user" {
			s.T().Log("Found work item link category in collection: ", *linkCatCollection.Data[i].Attributes.Name)
			toBeFound--
		}
	}
	require.Exactly(s.T(), 0, toBeFound, "Not all required work item link categories (system and user) where found.")
}

func getWorkItemLinkCategoryTestData(t *testing.T) []testSecureAPI {
	privatekey, err := jwt.ParseRSAPrivateKeyFromPEM((wilCatConfiguration.GetTokenPrivateKey()))
	if err != nil {
		t.Fatal("Could not parse Key ", err)
	}
	differentPrivatekey, err := jwt.ParseRSAPrivateKeyFromPEM(([]byte(RSADifferentPrivateKeyTest)))
	if err != nil {
		t.Fatal("Could not parse different private key ", err)
	}

	createWorkItemLinkCategoryPayloadString := bytes.NewBuffer([]byte(`
		{
			"data": {
				"attributes": {
					"description": "A sample work item link category",
					"name": "sample",
					"version": 0
				},
				"id": "6c5610be-30b2-4880-9fec-81e4f8e4fddd",
				"type": "workitemlinkcategories"
			}
		}
		`))

	return []testSecureAPI{
		// Create Work Item API with different parameters
		{
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPost,
			url:                endpointWorkItemLinkCategories,
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Update Work Item API with different parameters
		{
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodPatch,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            createWorkItemLinkCategoryPayloadString,
			jwtToken:           "",
		},
		// Delete Work Item API with different parameters
		{
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getExpiredAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getMalformedAuthHeader(t, privatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           getValidAuthHeader(t, differentPrivatekey),
		}, {
			method:             http.MethodDelete,
			url:                endpointWorkItemLinkCategories + "/6c5610be-30b2-4880-9fec-81e4f8e4fd76",
			expectedStatusCode: http.StatusUnauthorized,
			expectedErrorCode:  jsonapi.ErrorCodeJWTSecurityError,
			payload:            nil,
			jwtToken:           "",
		},
		// Try fetching a random work item link category
		// We do not have security on GET hence this should return 404 not found
		{
			method:             http.MethodGet,
			url:                endpointWorkItemLinkCategories + "/fc591f38-a805-4abd-bfce-2460e49d8cc4",
			expectedStatusCode: http.StatusNotFound,
			expectedErrorCode:  jsonapi.ErrorCodeNotFound,
			payload:            nil,
			jwtToken:           "",
		},
	}
}

// This test case will check authorized access to Create/Update/Delete APIs
func (s *workItemLinkCategorySuite) TestUnauthorizeWorkItemLinkCategoryCUD() {
	UnauthorizeCreateUpdateDeleteTest(s.T(), getWorkItemLinkCategoryTestData, func() *goa.Service {
		return goa.New("TestUnauthorizedCreateWorkItemLinkCategory-Service")
	}, func(service *goa.Service) error {
		controller := NewWorkItemLinkCategoryController(service, gormapplication.NewGormDB(s.db))
		app.MountWorkItemLinkCategoryController(service, controller)
		return nil
	})
}
