package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
)

func TestCompatibleFields(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := FieldDefinition{
		Label:       "a",
		Description: "description for 'a'",
		Required:    true,
		Type: ListType{
			SimpleType:    SimpleType{Kind: KindList},
			ComponentType: SimpleType{Kind: KindString},
		},
	}

	t.Run("compatible field definition", func(t *testing.T) {
		t.Parallel()
		// given
		b := FieldDefinition{
			Label:       "b",
			Description: "description for 'b'",
			Required:    true,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
		}
		// then
		assert.True(t, compatibleFields(a, b), "fields %+v and %+v are not detected as being compatible", a, b)
	})
	t.Run("incompatible field definition (incompatible fields)", func(t *testing.T) {
		t.Parallel()
		// given
		c := FieldDefinition{
			Label:       "c",
			Description: "description for 'c'",
			Required:    true,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindInteger},
			},
		}
		// then
		assert.False(t, compatibleFields(a, c), "fields %+v and %+v are not detected as being incompatible", a, c)
	})
	t.Run("incompatible field definition (different required field)", func(t *testing.T) {
		t.Parallel()
		// given
		d := FieldDefinition{
			Label:       "c",
			Description: "description for 'd'",
			Required:    false,
			Type: ListType{
				SimpleType:    SimpleType{Kind: KindList},
				ComponentType: SimpleType{Kind: KindString},
			},
		}
		// then
		assert.False(t, compatibleFields(a, d), "fields %+v and %+v are not detected as being incompatible", a, d)
	})
}
