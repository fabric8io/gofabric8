package workitem_test

import (
	"sync"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var cache = workitem.NewWorkItemTypeCache()

func TestGetNotExistingTypeReturnsNotOk(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get(uuid.FromStringOrNil("c88e6669-53f9-4aa1-be98-877b850daf88"))
	assert.False(t, ok)
}

func TestGetReturnsPreviouslyPutWIT(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	_, ok := cache.Get(uuid.FromStringOrNil("3566837f-aa98-4792-bce1-75c995d4e98c"))
	assert.False(t, ok)

	wit := workitem.WorkItemType{
		ID:   uuid.FromStringOrNil("3566837f-aa98-4792-bce1-75c995d4e98c"),
		Name: "testReadingWriting",
	}
	cache.Put(wit)

	cachedWit, ok := cache.Get(uuid.FromStringOrNil("3566837f-aa98-4792-bce1-75c995d4e98c"))
	assert.True(t, ok)
	assert.Equal(t, wit, cachedWit)
}

func TestGetReturnNotOkAfterClear(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	c := workitem.NewWorkItemTypeCache()
	c.Put(workitem.WorkItemType{
		ID:   uuid.FromStringOrNil("aa6ef831-36db-4e99-9e33-6f793472f769"),
		Name: "testClear",
	})
	_, ok := c.Get(uuid.FromStringOrNil("aa6ef831-36db-4e99-9e33-6f793472f769"))
	assert.True(t, ok)

	c.Clear()
	_, ok = c.Get(uuid.FromStringOrNil("aa6ef831-36db-4e99-9e33-6f793472f769"))
	assert.False(t, ok)
}

func TestNoFailuresWithConcurrentMapReadAndMapWrite(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	wit := workitem.WorkItemType{
		ID:   uuid.FromStringOrNil("68e90fa9-dba1-4448-99a4-ae70fb2b45f9"),
		Name: "testConcurrentAccess",
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 3000; i++ {
			cache.Put(wit)
		}
	}()
	cache.Put(wit)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			cachedWit, ok := cache.Get(uuid.FromStringOrNil("68e90fa9-dba1-4448-99a4-ae70fb2b45f9"))
			assert.True(t, ok)
			assert.Equal(t, wit, cachedWit)
		}
	}()
	wg.Wait()
}
