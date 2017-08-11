package link

import (
	"time"

	convert "github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLink represents the connection of two work items as it is stored in the db
type WorkItemLink struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Version for optimistic concurrency control
	Version    int
	SourceID   uuid.UUID `sql:"type:uuid"`
	TargetID   uuid.UUID `sql:"type:uuid"`
	LinkTypeID uuid.UUID `sql:"type:uuid"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLink{}
var _ convert.Equaler = (*WorkItemLink)(nil)

// Equal returns true if two WorkItemLink objects are equal; otherwise false is returned.
func (l WorkItemLink) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLink)
	if !ok {
		return false
	}
	if !l.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !uuid.Equal(l.ID, other.ID) {
		return false
	}
	if l.Version != other.Version {
		return false
	}
	if l.SourceID != other.SourceID {
		return false
	}
	if l.TargetID != other.TargetID {
		return false
	}
	if l.LinkTypeID != other.LinkTypeID {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link
// cannot be used for the creation of a new work item link.
func (l *WorkItemLink) CheckValidForCreation() error {
	if uuid.Equal(l.LinkTypeID, uuid.Nil) {
		return errors.NewBadParameterError("link_type_id", l.LinkTypeID)
	}
	return nil
}

// TableName implements gorm.tabler
func (l WorkItemLink) TableName() string {
	return "work_item_links"
}

// GetETagData returns the field values to use to generate the ETag
func (l WorkItemLink) GetETagData() []interface{} {
	return []interface{}{l.ID, l.Version}
}

// GetLastModified returns the last modification time
func (l WorkItemLink) GetLastModified() time.Time {
	return l.UpdatedAt
}
