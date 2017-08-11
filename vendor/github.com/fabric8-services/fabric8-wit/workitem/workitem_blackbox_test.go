package workitem_test

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestWorkItem_Equal(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	a := workitem.WorkItemStorage{
		ID:      uuid.NewV4(),
		Number:  1,
		Type:    uuid.NewV4(),
		Version: 0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID: space.SystemSpace,
	}

	// Test type difference
	b := convert.DummyEqualer{}
	assert.False(t, a.Equal(b))

	// Test lifecycle difference
	c := a
	c.Lifecycle = gormsupport.Lifecycle{CreatedAt: time.Now().Add(time.Duration(1000))}
	assert.False(t, a.Equal(c))

	// Test type difference
	d := a
	d.Type = uuid.NewV4()
	assert.False(t, a.Equal(d))

	// Test version difference
	e := a
	e.Version += 1
	assert.False(t, a.Equal(e))

	// Test version difference
	f := a
	f.Version += 1
	assert.False(t, a.Equal(f))

	// Test ID difference
	g := a
	g.ID = uuid.NewV4()
	assert.False(t, a.Equal(g))

	// Test Number difference
	h := a
	h.Number = 42
	assert.False(t, a.Equal(g))

	// Test fields difference
	i := a
	i.Fields = workitem.Fields{}
	assert.False(t, a.Equal(i))

	// Test Space
	j := a
	j.SpaceID = uuid.NewV4()
	assert.False(t, a.Equal(j))

	k := workitem.WorkItemStorage{
		ID:      a.ID,
		Type:    a.Type,
		Version: 0,
		Fields: workitem.Fields{
			"foo": "bar",
		},
		SpaceID: space.SystemSpace,
	}
	assert.True(t, a.Equal(k))
	assert.True(t, k.Equal(a))
}
