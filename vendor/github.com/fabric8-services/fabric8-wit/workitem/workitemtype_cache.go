package workitem

import (
	"sync"

	"github.com/fabric8-services/fabric8-wit/log"
	uuid "github.com/satori/go.uuid"
)

type witCacheMap map[uuid.UUID]WorkItemType

// WorkItemTypeCache represents WorkItemType cache
type WorkItemTypeCache struct {
	cache   witCacheMap
	mapLock sync.RWMutex
}

// NewWorkItemTypeCache constructs WorkItemTypeCache
func NewWorkItemTypeCache() *WorkItemTypeCache {
	witCache := WorkItemTypeCache{}
	witCache.cache = make(witCacheMap)
	return &witCache
}

// Get returns WorkItemType by ID.
// The second value (ok) is a bool that is true if the WorkItemType exists in the cache, and false if not.
func (c *WorkItemTypeCache) Get(id uuid.UUID) (WorkItemType, bool) {
	c.mapLock.RLock()
	defer c.mapLock.RUnlock()
	w, ok := c.cache[id]
	return w, ok
}

// Put puts a work item type to the cache
func (c *WorkItemTypeCache) Put(wit WorkItemType) {
	c.mapLock.Lock()
	defer c.mapLock.Unlock()
	c.cache[wit.ID] = wit
}

// Clear clears the cache
func (c *WorkItemTypeCache) Clear() {
	c.mapLock.Lock()
	defer c.mapLock.Unlock()
	log.Info(nil, nil, "Clearing work item cache")

	c.cache = make(witCacheMap)
}
