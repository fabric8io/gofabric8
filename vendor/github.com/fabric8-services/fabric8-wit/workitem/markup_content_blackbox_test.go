package workitem_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/stretchr/testify/assert"
)

func TestNewMarkupContentFromMapWithValidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[rendering.ContentKey] = "foo"
	input[rendering.MarkupKey] = rendering.SystemMarkupMarkdown
	// when
	result := rendering.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[rendering.ContentKey].(string), result.Content)
	assert.Equal(t, input[rendering.MarkupKey].(string), result.Markup)
}

func TestNewMarkupContentFromMapWithInvalidMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[rendering.ContentKey] = "foo"
	input[rendering.MarkupKey] = "bar"
	// when
	result := rendering.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[rendering.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithMissingMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[rendering.ContentKey] = "foo"
	// when
	result := rendering.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[rendering.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}

func TestNewMarkupContentFromMapWithEmptyMarkup(t *testing.T) {
	// given
	input := make(map[string]interface{})
	input[rendering.ContentKey] = "foo"
	input[rendering.MarkupKey] = ""
	// when
	result := rendering.NewMarkupContentFromMap(input)
	// then
	assert.Equal(t, input[rendering.ContentKey].(string), result.Content)
	assert.Equal(t, rendering.SystemMarkupDefault, result.Markup)
}
