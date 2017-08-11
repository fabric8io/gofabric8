package link_test

import (
	"testing"

	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

// TestWorkItemType_Equal Tests equality of two work item link categories
func TestWorkItemLinkCategory_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	description := "An example description"
	a := link.WorkItemLinkCategory{
		ID:          uuid.FromStringOrNil("0e671e36-871b-43a6-9166-0c4bd573e231"),
		Name:        "Example work item link category",
		Description: &description,
		Version:     0,
	}

	// Test types
	b := convert.DummyEqualer{}
	require.False(t, a.Equal(b))

	// Test lifecycle
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(c))

	// Test version
	c = a
	c.Version += 1
	require.False(t, a.Equal(c))

	// Test name
	c = a
	c.Name = "bar"
	require.False(t, a.Equal(c))

	// Test description
	otherDescription := "bar"
	c = a
	c.Description = &otherDescription
	require.False(t, a.Equal(c))

	// Test equality
	c = a
	require.True(t, a.Equal(c))

	// Test ID
	c = a
	c.ID = uuid.FromStringOrNil("33371e36-871b-43a6-9166-0c4bd573e333")
	require.False(t, a.Equal(c))

	// Test when one Description is nil
	c = a
	c.Description = nil
	require.False(t, a.Equal(c))
}
