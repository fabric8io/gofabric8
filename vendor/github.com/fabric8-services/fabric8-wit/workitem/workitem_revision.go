package workitem

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// RevisionType defines the type of revision for a work item
type RevisionType int

const (
	_ RevisionType = iota // ignore first value by assigning to blank identifier
	// RevisionTypeCreate a work item creation
	RevisionTypeCreate // 1
	// RevisionTypeDelete a work item deletion
	RevisionTypeDelete // 2
	_                  // ignore 3rd value
	// RevisionTypeUpdate a work item update
	RevisionTypeUpdate // 4
)

// Revision represents a version of a work item
type Revision struct {
	ID uuid.UUID `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type RevisionType `gorm:"column:revision_type"`
	// the identity of author of the workitem modification
	ModifierIdentity uuid.UUID `sql:"type:uuid" gorm:"column:modifier_id"`
	// the id of the work item that changed
	WorkItemID uuid.UUID `gorm:"column:work_item_id"`
	// Id of the type of this work item
	WorkItemTypeID uuid.UUID `gorm:"column:work_item_type_id"`
	// Version of the workitem that was modified
	WorkItemVersion int `gorm:"column:work_item_version"`
	// the field values (or empty when the work item was deleted)
	WorkItemFields Fields `gorm:"column:work_item_fields" sql:"type:jsonb"`
}

const (
	revisionTableName = "work_item_revisions"
)

// TableName implements gorm.tabler
func (w Revision) TableName() string {
	return revisionTableName
}
