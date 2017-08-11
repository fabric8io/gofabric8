package workitem_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

// TestJsonMarshalListType constructs a work item type, writes it to JSON (marshalling),
// and converts it back from JSON into a work item type (unmarshalling)
func TestJsonMarshalListType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	lt := workitem.ListType{
		SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
		ComponentType: workitem.SimpleType{Kind: workitem.KindInteger},
	}

	field := workitem.FieldDefinition{
		Type:     lt,
		Required: false,
	}

	expectedWIT := workitem.WorkItemType{
		Name: "first type",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": field},
	}

	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT workitem.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestMarshalEnumType(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := workitem.EnumType{
		SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	fd := workitem.FieldDefinition{
		Type:     et,
		Required: true,
	}

	desc := "some description"
	expectedWIT := workitem.WorkItemType{
		Name:        "first type",
		Description: &desc,
		Fields: map[string]workitem.FieldDefinition{
			"aListType": fd},
	}
	bytes, err := json.Marshal(expectedWIT)
	if err != nil {
		t.Error(err)
	}

	var parsedWIT workitem.WorkItemType
	json.Unmarshal(bytes, &parsedWIT)

	if !expectedWIT.Equal(parsedWIT) {
		t.Errorf("Unmarshalled work item type: \n %v \n has not the same type as \"normal\" workitem type: \n %v \n", parsedWIT, expectedWIT)
	}
}

func TestWorkItemType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	fd := workitem.FieldDefinition{
		Type: workitem.EnumType{
			SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
			Values:     []interface{}{"open", "done", "closed"},
		},
		Required: true,
	}

	desc := "some description"
	a := workitem.WorkItemType{
		Name:        "foo",
		Description: &desc,
		Icon:        "fa-bug",
		Fields: map[string]workitem.FieldDefinition{
			"aListType": fd,
		},
	}

	// Test types
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test version
	d := a
	d.Version += 1
	assert.False(t, a.Equal(d))

	// Test name
	e := a
	e.Name = "bar"
	assert.False(t, a.Equal(e))

	// Test parent path
	f := a
	f.Path = "foobar"
	assert.False(t, a.Equal(f))

	// Test field array length
	g := a
	g.Fields = map[string]workitem.FieldDefinition{}
	assert.False(t, a.Equal(g))

	// Test field key existence
	h := workitem.WorkItemType{
		Name: "foo",
		Fields: map[string]workitem.FieldDefinition{
			"bar": fd,
		},
	}
	assert.False(t, a.Equal(h))

	// Test field difference
	i := workitem.WorkItemType{
		Name:        "foo",
		Description: &desc,
		Fields: map[string]workitem.FieldDefinition{
			"aListType": {
				Type: workitem.EnumType{
					SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
					Values:     []interface{}{"open", "done", "closed"},
				},
				Required: false,
			},
		},
	}
	assert.False(t, a.Equal(i))

	// Test description
	j := a
	otherDesc := "some other description"
	j.Description = &otherDesc
	assert.False(t, a.Equal(j))

	// Test icon
	j = a
	j.Icon = "fa-cog"
	assert.False(t, a.Equal(j))
}

func TestMarshalFieldDef(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	et := workitem.EnumType{
		SimpleType: workitem.SimpleType{workitem.KindEnum},
		Values:     []interface{}{"open", "done", "closed"},
	}
	expectedFieldDef := workitem.FieldDefinition{
		Type:     et,
		Required: true,
	}

	bytes, err := json.Marshal(expectedFieldDef)
	if err != nil {
		t.Error(err)
	}

	var parsedFieldDef workitem.FieldDefinition
	json.Unmarshal(bytes, &parsedFieldDef)
	if !expectedFieldDef.Equal(parsedFieldDef) {
		t.Errorf("Unmarshalled field definition: \n %v \n has not the same type as \"normal\" field definition: \n %v \n", parsedFieldDef, expectedFieldDef)
	}
}

func TestMarshalArray(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	original := []interface{}{float64(1), float64(2), float64(3)}
	bytes, err := json.Marshal(original)
	if err != nil {
		t.Error(err)
	}
	var read []interface{}
	json.Unmarshal(bytes, &read)
	if !reflect.DeepEqual(original, read) {
		fmt.Printf("cap=[%d, %d], len=[%d, %d]\n", cap(original), cap(read), len(original), len(read))
		t.Error("not equal")
	}
}

func TestWorkItemTypeIsTypeOrSubtypeOf(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Prepare some UUIDs for use in tests
	id1 := uuid.FromStringOrNil("68e90fa9-dba1-4448-99a4-ae70fb2b45f9")
	id2 := uuid.FromStringOrNil("aa6ef831-36db-4e99-9e33-6f793472f769")
	id3 := uuid.FromStringOrNil("3566837f-aa98-4792-bce1-75c995d4e98c")
	id4 := uuid.FromStringOrNil("c88e6669-53f9-4aa1-be98-877b850daf88")
	// Prepare the ltree nodes based on the IDs
	node1 := workitem.LtreeSafeID(id1)
	node2 := workitem.LtreeSafeID(id2)
	node3 := workitem.LtreeSafeID(id3)

	// Test types and subtypes
	assert.True(t, workitem.WorkItemType{ID: id1, Path: node1}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id2, Path: node1 + "." + node2}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id2, Path: node1 + "." + node2}.IsTypeOrSubtypeOf(id2))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id1))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id2))
	assert.True(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id3))

	// Test we actually do return false someNodees
	assert.False(t, workitem.WorkItemType{ID: id3, Path: node1 + "." + node2 + "." + node3}.IsTypeOrSubtypeOf(id4))
	assert.False(t, workitem.WorkItemType{ID: id1, Path: node1}.IsTypeOrSubtypeOf(id4))
}
