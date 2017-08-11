package link

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// RevisionType defines the type of revision for a work item link
type RevisionType int

const (
	_ RevisionType = iota // ignore first value by assigning to blank identifier
	// RevisionTypeCreate a work item link creation
	RevisionTypeCreate // 1
	// RevisionTypeDelete a work item link deletion
	RevisionTypeDelete // 2
	_                  // ignore 3rd value
	// RevisionTypeUpdate a work item link update
	RevisionTypeUpdate // 4
)

// Revision represents a version of a work item link
type Revision struct {
	ID uuid.UUID `gorm:"primary_key"`
	// the timestamp of the modification
	Time time.Time `gorm:"column:revision_time"`
	// the type of modification
	Type RevisionType `gorm:"column:revision_type"`
	// the identity of author of the work item modification
	ModifierIdentity uuid.UUID `sql:"type:uuid" gorm:"column:modifier_id"`
	// the ID of the work item link that changed
	WorkItemLinkID uuid.UUID `sql:"type:uuid"`
	// the version of the work item link that changed
	WorkItemLinkVersion int
	// the ID of the source of the work item link that changed
	WorkItemLinkSourceID uuid.UUID `sql:"type:uuid"`
	// the ID of the target of the work item link that changed
	WorkItemLinkTargetID uuid.UUID `sql:"type:uuid"`
	// the ID of the type of the work item link that changed
	WorkItemLinkTypeID uuid.UUID `sql:"type:uuid"`
}

const (
	revisionTableName = "work_item_link_revisions"
)

// TableName implements gorm.tabler
func (w Revision) TableName() string {
	return revisionTableName
}
