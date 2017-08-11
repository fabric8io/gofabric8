package workitem

import (
	"time"

	"github.com/fabric8-services/fabric8-wit/log"

	uuid "github.com/satori/go.uuid"
)

// WorkItem the model structure for the work item.
type WorkItem struct {
	// unique id per installation (used for references at the DB level)
	ID uuid.UUID
	// unique number per _space_
	Number int
	// ID of the type of this work item
	Type uuid.UUID
	// Version for optimistic concurrency control
	Version int
	// ID of the space to which this work item belongs
	SpaceID uuid.UUID
	// The field values, according to the field type
	Fields map[string]interface{}
	// optional, private timestamp of the latest addition/removal of a relationship with this workitem
	// this field is used to generate the `ETag` and `Last-Modified` values in the HTTP responses and conditional requests processing
	relationShipsChangedAt *time.Time
}

// WICountsPerIteration counting work item states by iteration
type WICountsPerIteration struct {
	IterationID string `gorm:"column:iterationid"`
	Total       int
	Closed      int
}

// GetETagData returns the field values to use to generate the ETag
func (wi WorkItem) GetETagData() []interface{} {
	return []interface{}{wi.ID, wi.Version, wi.relationShipsChangedAt}
}

// GetLastModified returns the last modification time
func (wi WorkItem) GetLastModified() time.Time {
	var lastModified *time.Time // default value
	if updatedAt, ok := wi.Fields[SystemUpdatedAt].(time.Time); ok {
		lastModified = &updatedAt
	}
	// also check the optional 'relationShipsChangedAt' field
	if wi.relationShipsChangedAt != nil && (lastModified == nil || wi.relationShipsChangedAt.After(*lastModified)) {
		lastModified = wi.relationShipsChangedAt
	}

	log.Debug(nil, map[string]interface{}{"wi_id": wi.ID}, "Last modified value: %v", lastModified)
	return *lastModified
}
