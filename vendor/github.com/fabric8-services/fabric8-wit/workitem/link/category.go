package link

import (
	convert "github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	uuid "github.com/satori/go.uuid"
)

// Never ever change these UUIDs!!!
var (
	SystemWorkItemLinkCategorySystemID = uuid.FromStringOrNil("B1482C65-A64D-4058-BEB0-62F7198CB0F4")
	SystemWorkItemLinkCategoryUserID   = uuid.FromStringOrNil("2F24724F-797C-4073-8B16-4BB8CE9E84A6")
)

// WorkItemLinkCategory represents the category of a work item link as it is stored in the db
type WorkItemLinkCategory struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Name is the unique name of this work item link category.
	Name string
	// Description is an optional description of the work item link category
	Description *string
	// Version for optimistic concurrency control
	Version int
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkCategory{}
var _ convert.Equaler = (*WorkItemLinkCategory)(nil)

// Equal returns true if two WorkItemLinkCategory objects are equal; otherwise false is returned.
func (c WorkItemLinkCategory) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkCategory)
	if !ok {
		return false
	}
	if !c.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if c.ID != other.ID {
		return false
	}
	if c.Name != other.Name {
		return false
	}
	if c.Version != other.Version {
		return false
	}
	if !strPtrIsNilOrContentIsEqual(c.Description, other.Description) {
		return false
	}
	return true
}

// TableName implements gorm.tabler
func (c WorkItemLinkCategory) TableName() string {
	return "work_item_link_categories"
}
