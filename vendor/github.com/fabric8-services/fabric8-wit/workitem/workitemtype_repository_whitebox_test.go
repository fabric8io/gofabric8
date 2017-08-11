package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/stretchr/testify/assert"
)

func TestConvertAnyToKind(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, err := ConvertAnyToKind(1234)
	assert.NotNil(t, err)
}

func TestConvertStringToKind(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, err := ConvertStringToKind("DefinitivelyNotAKind")
	assert.NotNil(t, err)
}
