package workitem

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
)

func TestEnumTypeContains(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	haystack := []interface{}{1, 2, 3, 4}

	// Check for existence
	needle := interface{}(3)
	assert.True(t, contains(haystack, needle))

	// Check for absence
	needle = interface{}(42)
	assert.False(t, contains(haystack, needle))
}
