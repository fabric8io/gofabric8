package path_test

import (
	"strings"
	"testing"

	"fmt"

	"github.com/fabric8-services/fabric8-wit/path"
	"github.com/fabric8-services/fabric8-wit/resource"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEmpty(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	lp := path.Path{}
	require.True(t, lp.IsEmpty())
}

func TestThis(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	greatGrandParent := uuid.NewV4()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{greatGrandParent, grandParent, immediateParent}
	assert.Equal(t, immediateParent, lp.This())

	lp2 := path.Path{}
	require.Equal(t, uuid.Nil, lp2.This())
}

func TestConvert(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	expected := fmt.Sprintf("%s.%s", grandParent, immediateParent)
	expected = strings.Replace(expected, "-", "_", -1)
	assert.Equal(t, expected, lp.Convert())

	lp2 := path.Path{}
	require.Empty(t, lp2.Convert())
}

func TestToString(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	expected := fmt.Sprintf("/%s/%s", grandParent, immediateParent)
	require.Equal(t, expected, lp.String())

	lp2 := path.Path{}
	require.Equal(t, path.SepInService, lp2.String())
}

func TestRoot(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	assert.Equal(t, path.Path{grandParent}, lp.Root())

	lp2 := path.Path{}
	assert.Equal(t, path.Path{uuid.Nil}, lp2.Root())
}

func TestParent(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	require.Equal(t, path.Path{immediateParent}, lp.Parent())

	lp2 := path.Path{}
	require.Equal(t, path.Path{uuid.Nil}, lp2.Parent())
}

func TestValuerImplementation(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()

	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	expected := fmt.Sprintf("%s.%s", grandParent, immediateParent)
	expected = strings.Replace(expected, "-", "_", -1)
	v, err := lp.Value()
	require.Nil(t, err)
	assert.Equal(t, expected, v)
}

func TestScannerImplementation(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	grandParent := uuid.NewV4()
	immediateParent := uuid.NewV4()
	lp := path.Path{grandParent, immediateParent}
	v, err := lp.Value()
	require.Nil(t, err)

	lp2 := path.Path{}
	err2 := lp2.Scan([]byte(v.(string)))
	require.Nil(t, err2)
	require.Len(t, lp2, 2)
	assert.Equal(t, lp2, lp)
	assert.Equal(t, lp[0], lp2[0])

	lp3 := path.Path{}
	err3 := lp2.Scan(nil)
	require.Nil(t, err3)
	assert.Len(t, lp3, 0)
}

func TestToExpression(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	uuid1 := uuid.NewV4()
	uuid2 := uuid.NewV4()
	uuid3 := uuid.NewV4()
	actual := path.ToExpression(path.Path{uuid1, uuid2}, uuid3)

	expected := fmt.Sprintf("%s.%s.%s", uuid1, uuid2, uuid3)
	expected = strings.Replace(expected, "-", "_", -1)

	assert.Equal(t, expected, actual)
}
