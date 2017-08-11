package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
)

func TestSimpleTypeEqual(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test type difference
	a := SimpleType{Kind: KindString}
	assert.False(t, a.Equal(convert.DummyEqualer{}))

	// Test kind difference
	b := SimpleType{Kind: KindInteger}
	assert.False(t, a.Equal(b))
}

func TestConvertToModel(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	// Test nil value
	a := SimpleType{Kind: KindString}
	res, err := a.ConvertToModel(nil)
	assert.Nil(t, res)
	assert.Nil(t, err)

	// Test default case in swtich statement
	b := 42
	res, err = a.ConvertToModel(&b)
	assert.NotNil(t, err)
	assert.Nil(t, res)
}
