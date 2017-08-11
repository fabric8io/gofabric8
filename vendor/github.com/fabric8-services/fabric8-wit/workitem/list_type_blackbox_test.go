package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
)

func TestListType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}

	// Test type incompatibility
	assert.False(t, a.Equal(convert.DummyEqualer{}))

	// Test simple type difference
	b := ListType{
		SimpleType:    SimpleType{Kind: KindString},
		ComponentType: SimpleType{Kind: KindString},
	}
	assert.False(t, a.Equal(b))

	// Test component type difference
	c := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindInteger},
	}
	assert.False(t, a.Equal(c))

	// Test equality
	d := ListType{
		SimpleType:    SimpleType{Kind: KindList},
		ComponentType: SimpleType{Kind: KindString},
	}
	assert.True(t, d.Equal(a))
	assert.True(t, a.Equal(d)) // test the inverse
}
