package cmds

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEmptyAdaptFabric8ImagesInResourceDescriptor(t *testing.T) {
	t.Log("Hello ")
	data := []byte("{}")
	json, err := adaptFabric8ImagesInResourceDescriptor(data, "", "arm")
	assert.Nil(t, err, "No error should be set")
	assert.Equal(t, data, json, "Nothing should be replaced")
}

func TestAdaptFabric8ImagesInResourceDescriptorArm(t *testing.T) {
	testParams := []string{
		"", "amd64",
		"registry", "arm",
		"registry", "amd64",
		"", "arm",
	}

	testDataSet := [][]string{
		{
			"fabric8/test-image:1.0", "fabric8/test-image:1.0",
			"bla/test-image:1.0", "bla/test-image:1.0",
			"fabric8/test-image", "fabric8/test-image",
		},
		{
			"fabric8/test-image:1.0", "registry/fabric8/test-image-arm:1.0",
			"bla/test-image:1.0", "bla/test-image:1.0",
			"fabric8/test-image", "registry/fabric8/test-image-arm",
		},
		{
			"fabric8/test-image:1.0", "registry/fabric8/test-image:1.0",
			"bla/test-image:1.0", "bla/test-image:1.0",
			"fabric8/test-image", "registry/fabric8/test-image",
		},
		{
			"fabric8/test-image:1.0", "fabric8/test-image-arm:1.0",
			"bla/test-image:1.0", "bla/test-image:1.0",
			"fabric8/test-image", "fabric8/test-image-arm",
		},
	}
	for i := 0; i < len(testParams); i += 2 {
		registry := testParams[i]
		arch := testParams[i+1]

		testData := testDataSet[i/2]
		for j := 0; j < len(testData); j += 2 {
			data := "\"image\" : \"" + testData[j] + "\""
			expected := "\"image\" : \"" + testData[j+1] + "\""
			json, err := adaptFabric8ImagesInResourceDescriptor([]byte(data), registry, arch)
			assert.Nil(t, err, "Error should not be set")
			assert.Equal(t, expected, string(json[:]), "Image must match")
		}
	}
}
