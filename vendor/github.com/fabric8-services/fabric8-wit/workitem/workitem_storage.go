package workitem

import (
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	uuid "github.com/satori/go.uuid"
)

// WorkItemStorage represents a work item as it is stored in the database
type WorkItemStorage struct {
	gormsupport.Lifecycle
	// unique id per installation (used for references at the DB level)
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// unique number per _space_
	Number int
	// Id of the type of this work item
	Type uuid.UUID `sql:"type:uuid"`
	// Version for optimistic concurrency control
	Version int
	// the field values
	Fields Fields `sql:"type:jsonb"`
	// the position of workitem
	ExecutionOrder float64
	// Reference to one Space
	SpaceID uuid.UUID `sql:"type:uuid"`
	// optional timestamp of the latest addition/removal of a relationship with this workitem
	RelationShipsChangedAt *time.Time `sql:"column:relationships_changed_at"`
}

const (
	workitemTableName = "work_items"
)

// TableName implements gorm.tabler
func (wi WorkItemStorage) TableName() string {
	return workitemTableName
}

// Ensure WorkItem implements the Equaler interface
var _ convert.Equaler = WorkItemStorage{}
var _ convert.Equaler = (*WorkItemStorage)(nil)

// Equal returns true if two WorkItem objects are equal; otherwise false is returned.
func (wi WorkItemStorage) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemStorage)
	if !ok {
		return false
	}
	if !wi.Lifecycle.Equal(other.Lifecycle) {
		return false
	}

	if !uuid.Equal(wi.Type, other.Type) {
		return false
	}
	if wi.ID != other.ID {
		return false
	}
	if wi.Version != other.Version {
		return false
	}
	if wi.ExecutionOrder != other.ExecutionOrder {
		return false
	}
	if wi.SpaceID != other.SpaceID {
		return false
	}
	return wi.Fields.Equal(other.Fields)
}

// ParseWorkItemIDToUint64 does what it says
func ParseWorkItemIDToUint64(wiIDStr string) (uint64, error) {
	wiID, err := strconv.ParseUint(wiIDStr, 10, 64)
	if err != nil {
		return 0, errors.NewNotFoundError("work item ID", wiIDStr)
	}
	return wiID, nil
}
