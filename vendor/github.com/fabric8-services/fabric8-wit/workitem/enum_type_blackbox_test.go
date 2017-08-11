package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
)

func TestEnumType_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	stEnum := workitem.SimpleType{Kind: workitem.KindEnum}
	a := workitem.EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo", "bar"},
	}

	// Test type inequality
	assert.False(t, a.Equal(convert.DummyEqualer{}))

	// Test simple type difference
	stInteger := workitem.SimpleType{Kind: workitem.KindInteger}
	b := workitem.EnumType{
		SimpleType: workitem.SimpleType{Kind: workitem.KindInteger},
		BaseType:   stInteger,
	}
	assert.False(t, a.Equal(b))

	// Test base type difference
	c := workitem.EnumType{
		BaseType: stInteger,
	}
	assert.False(t, a.Equal(c))

	// Test values difference
	d := workitem.EnumType{
		BaseType: stEnum,
		Values:   []interface{}{"foo1", "bar2"},
	}
	assert.False(t, a.Equal(d))
}
