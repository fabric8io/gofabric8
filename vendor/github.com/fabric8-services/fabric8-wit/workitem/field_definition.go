package workitem

import (
	"encoding/json"
	"fmt"
	"reflect"

	"strings"

	"github.com/fabric8-services/fabric8-wit/convert"

	errs "github.com/pkg/errors"
)

// constants for describing possible field types
const (
	KindString            Kind = "string"
	KindInteger           Kind = "integer"
	KindFloat             Kind = "float"
	KindInstant           Kind = "instant"
	KindDuration          Kind = "duration"
	KindURL               Kind = "url"
	KindIteration         Kind = "iteration"
	KindWorkitemReference Kind = "workitem"
	KindUser              Kind = "user"
	KindEnum              Kind = "enum"
	KindList              Kind = "list"
	KindMarkup            Kind = "markup"
	KindArea              Kind = "area"
	KindCodebase          Kind = "codebase"
)

// Kind is the kind of field type
type Kind string

// IsSimpleType returns 'true' if the kind is simple, i.e., not a list nor an enum
func (k Kind) IsSimpleType() bool {
	return k != KindEnum && k != KindList
}

// FieldType describes the possible values of a FieldDefinition
type FieldType interface {
	GetKind() Kind
	// ConvertToModel converts a field value for use in the persistence layer
	ConvertToModel(value interface{}) (interface{}, error)
	// ConvertFromModel converts a field value for use in the REST API layer
	ConvertFromModel(value interface{}) (interface{}, error)
	// Implement the Equaler interface
	Equal(u convert.Equaler) bool
}

// FieldDefinition describes type & other restrictions of a field
type FieldDefinition struct {
	Required    bool
	Label       string
	Description string
	Type        FieldType
}

// Ensure FieldDefinition implements the Equaler interface
var _ convert.Equaler = FieldDefinition{}
var _ convert.Equaler = (*FieldDefinition)(nil)

// Equal returns true if two FieldDefinition objects are equal; otherwise false is returned.
func (f FieldDefinition) Equal(u convert.Equaler) bool {
	other, ok := u.(FieldDefinition)
	if !ok {
		return false
	}
	if f.Required != other.Required {
		return false
	}
	if f.Label != other.Label {
		return false
	}
	if f.Description != other.Description {
		return false
	}
	return f.Type.Equal(other.Type)
}

// ConvertToModel converts a field value for use in the persistence layer
func (f FieldDefinition) ConvertToModel(name string, value interface{}) (interface{}, error) {
	if f.Required && (value == nil || (f.Type.GetKind() == KindString && strings.TrimSpace(value.(string)) == "")) {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return f.Type.ConvertToModel(value)
}

// ConvertFromModel converts a field value for use in the REST API layer
func (f FieldDefinition) ConvertFromModel(name string, value interface{}) (interface{}, error) {
	if f.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return f.Type.ConvertFromModel(value)
}

type rawFieldDef struct {
	Required    bool
	Label       string
	Description string
	Type        *json.RawMessage
}

// Ensure rawFieldDef implements the Equaler interface
var _ convert.Equaler = rawFieldDef{}
var _ convert.Equaler = (*rawFieldDef)(nil)

// Equal returns true if two rawFieldDef objects are equal; otherwise false is returned.
func (f rawFieldDef) Equal(u convert.Equaler) bool {
	other, ok := u.(rawFieldDef)
	if !ok {
		return false
	}
	if f.Required != other.Required {
		return false
	}
	if f.Label != other.Label {
		return false
	}
	if f.Description != other.Description {
		return false
	}
	if f.Type == nil && other.Type == nil {
		return true
	}
	if f.Type != nil && other.Type != nil {
		return reflect.DeepEqual(f.Type, other.Type)
	}
	return false
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (f *FieldDefinition) UnmarshalJSON(bytes []byte) error {
	temp := rawFieldDef{}

	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return errs.WithStack(err)
	}
	rawType := map[string]interface{}{}
	json.Unmarshal(*temp.Type, &rawType)
	kind, err := ConvertAnyToKind(rawType["Kind"])

	if err != nil {
		return errs.WithStack(err)
	}
	switch *kind {
	case KindList:
		theType := ListType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, Label: temp.Label, Description: temp.Description}
	case KindEnum:
		theType := EnumType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, Label: temp.Label, Description: temp.Description}
	default:
		theType := SimpleType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errs.WithStack(err)
		}
		*f = FieldDefinition{Type: theType, Required: temp.Required, Label: temp.Label, Description: temp.Description}
	}
	return nil
}

func ConvertAnyToKind(any interface{}) (*Kind, error) {
	k, ok := any.(string)
	if !ok {
		return nil, fmt.Errorf("kind is not a string value %v", any)
	}
	return ConvertStringToKind(k)
}

func ConvertStringToKind(k string) (*Kind, error) {
	kind := Kind(k)
	switch kind {
	case KindString, KindInteger, KindFloat, KindInstant, KindDuration, KindURL, KindWorkitemReference, KindUser, KindEnum, KindList, KindIteration, KindMarkup, KindArea, KindCodebase:
		return &kind, nil
	}
	return nil, fmt.Errorf("kind '%s' is not a simple type", k)
}

// compatibleFields returns true if the existing and new field are compatible;
// otherwise false is returned. It does so by comparing all members of the field
// definition except for the label and description.
func compatibleFields(existing FieldDefinition, new FieldDefinition) bool {
	if existing.Required != new.Required {
		return false
	}
	return reflect.DeepEqual(existing.Type, new.Type)
}
