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

// TestWorkItemLink_Equal Tests equality of two work item links
func TestWorkItemLink_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   uuid.NewV4(),
		TargetID:   uuid.NewV4(),
		LinkTypeID: uuid.NewV4(),
	}

	// Test equality
	b := a
	require.True(t, a.Equal(b))

	// Test types
	c := convert.DummyEqualer{}
	require.False(t, a.Equal(c))

	// Test lifecycle
	b = a
	b.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	require.False(t, a.Equal(b))

	// Test ID
	b = a
	b.ID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test Version
	b = a
	b.Version += 1
	require.False(t, a.Equal(b))

	// Test SourceID
	b = a
	b.SourceID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test TargetID
	b = a
	b.TargetID = uuid.NewV4()
	require.False(t, a.Equal(b))

	// Test LinkTypeID
	b = a
	b.LinkTypeID = uuid.NewV4()
	require.False(t, a.Equal(b))
}

func TestWorkItemLinkCheckValidForCreation(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := link.WorkItemLink{
		ID:         uuid.NewV4(),
		SourceID:   uuid.NewV4(),
		TargetID:   uuid.NewV4(),
		LinkTypeID: uuid.NewV4(),
	}

	// Check valid
	b := a
	require.Nil(t, b.CheckValidForCreation())

	// Check empty LinkTypeID
	b = a
	b.LinkTypeID = uuid.Nil
	require.NotNil(t, b.CheckValidForCreation())
}
