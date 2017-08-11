package remoteworkitem

import (
	"encoding/json"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenMap(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	testString := []byte(`
		{
			"admins": [
				{
					"name": "aslak"
				}
			],
			"name": "shoubhik",
			"assignee": {
				"fixes": 2,
				"complete": true,
				"foo": [1, 2, 3, 4],
				"1": "sbose",
				"2": "pranav",
				"participants": {
					"4": "sbose56",
					"5": "sbose78"
				}
			}
		}`)
	var nestedMap map[string]interface{}
	err := json.Unmarshal(testString, &nestedMap)
	require.Nil(t, err)
	// when
	oneLevelMap := make(map[string]interface{})
	flatten(oneLevelMap, nestedMap, nil)
	// then
	// Test for string
	assert.Equal(t, oneLevelMap["assignee.participants.4"], "sbose56", "Incorrect mapping found for assignee.participants.4")
	// test for int
	assert.Equal(t, int(oneLevelMap["assignee.fixes"].(float64)), 2)
	// test for array
	assert.Equal(t, oneLevelMap["assignee.foo.0"], float64(1))
	// test for boolean
	assert.Equal(t, oneLevelMap["assignee.complete"], true)
	// test for array of object(s)
	assert.Equal(t, oneLevelMap["admins.0.name"], "aslak")
}

func TestConvertArrayToMap(t *testing.T) {
	// given
	resource.Require(t, resource.UnitTest)
	testArray := []interface{}{1, 2, 3, 4}
	// when
	testMap := convertArrayToMap(testArray)
	// then
	assert.Equal(t, testMap["0"], 1)
	assert.Equal(t, testMap["1"], 2)
	assert.Equal(t, testMap["2"], 3)
	assert.Equal(t, testMap["3"], 4)
}
