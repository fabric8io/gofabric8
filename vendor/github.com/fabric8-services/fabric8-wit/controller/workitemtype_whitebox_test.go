package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTypeFromModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	// given

	//------------------------------
	// Work item type in model space
	//------------------------------

	descFoo := "Description of 'foo'"
	id := uuid.NewV4()
	createdAt := time.Now().Add(-1 * time.Hour).UTC()
	updatedAt := time.Now().UTC()
	a := workitem.WorkItemType{
		ID: id,
		Lifecycle: gormsupport.Lifecycle{
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:        "foo",
		Description: &descFoo,
		Version:     42,
		Path:        "something",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": {
				Label:       "some list type",
				Description: "description for 'some list type'",
				Type: workitem.EnumType{
					BaseType:   workitem.SimpleType{Kind: workitem.KindString},
					SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: true,
			},
		},
		SpaceID: space.SystemSpace,
	}
	//----------------------------
	// Work item type in app space
	//----------------------------

	// Create an enumeration of animal names
	typeStrings := []string{"open", "done", "closed"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"
	reqLong := &goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}
	spaceSelfURL := rest.AbsoluteURL(reqLong, app.SpaceHref(space.SystemSpace.String()))
	version := 42
	expected := app.WorkItemTypeSingle{
		Data: &app.WorkItemTypeData{
			ID:   &id,
			Type: "workitemtypes",
			Attributes: &app.WorkItemTypeAttributes{
				Name:        "foo",
				Description: &descFoo,
				Version:     &version,
				CreatedAt:   &createdAt,
				UpdatedAt:   &updatedAt,
				Fields: map[string]*app.FieldDefinition{
					"aListType": {
						Required:    true,
						Label:       "some list type",
						Description: "description for 'some list type'",
						Type: &app.FieldType{
							BaseType: &stString,
							Kind:     "enum",
							Values:   typeEnum,
						},
					},
				},
			},
			Relationships: &app.WorkItemTypeRelationships{
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
	// when
	result := ConvertWorkItemTypeFromModel(reqLong, &a)
	// then
	require.NotNil(t, result.ID)
	assert.True(t, uuid.Equal(*expected.Data.ID, *result.ID))
	assert.Equal(t, expected.Data.Attributes.Version, result.Attributes.Version)
	assert.Equal(t, expected.Data.Attributes.CreatedAt, result.Attributes.CreatedAt)
	assert.Equal(t, expected.Data.Attributes.UpdatedAt, result.Attributes.UpdatedAt)
	assert.Equal(t, expected.Data.Attributes.Name, result.Attributes.Name)
	require.NotNil(t, result.Attributes.Description)
	assert.Equal(t, *expected.Data.Attributes.Description, *result.Attributes.Description)
	assert.Len(t, result.Attributes.Fields, len(expected.Data.Attributes.Fields))
	assert.Equal(t, expected.Data.Attributes.Fields, result.Attributes.Fields)
}

func TestConvertFieldTypes(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	types := []workitem.FieldType{
		workitem.SimpleType{Kind: workitem.KindInteger},
		workitem.ListType{workitem.SimpleType{Kind: workitem.KindList}, workitem.SimpleType{Kind: workitem.KindString}},
		workitem.EnumType{workitem.SimpleType{Kind: workitem.KindEnum}, workitem.SimpleType{Kind: workitem.KindString}, []interface{}{"foo", "bar"}},
	}

	for _, theType := range types {
		t.Logf("testing type %v", theType)
		if err := testConvertFieldType(theType); err != nil {
			t.Error(err.Error())
		}
	}
}

func testConvertFieldType(original workitem.FieldType) error {
	converted := convertFieldTypeFromModel(original)
	reconverted, _ := convertFieldTypeToModel(converted)
	if !reflect.DeepEqual(original, reconverted) {
		return fmt.Errorf("reconverted should be %v, but is %v", original, reconverted)
	}
	return nil
}

func TestConvertFieldTypeToModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Create an enumeration of animal names
	typeStrings := []string{"open", "done", "closed"}

	// Convert string slice to slice of interface{} in O(n) time.
	typeEnum := make([]interface{}, len(typeStrings))
	for i := range typeStrings {
		typeEnum[i] = typeStrings[i]
	}

	// Create the type for "animal-type" field based on the enum above
	stString := "string"

	_ = &app.FieldType{
		BaseType: &stString,
		Kind:     "DefinitivelyNotAType",
		Values:   typeEnum,
	}
	_, err := convertFieldTypeToModel(app.FieldType{Kind: "DefinitivelyNotAType"})
	assert.NotNil(t, err)
}
