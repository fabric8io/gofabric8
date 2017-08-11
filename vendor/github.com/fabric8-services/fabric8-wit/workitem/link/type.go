package link

import (
	"time"

	convert "github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	TopologyNetwork         = "network"
	TopologyDirectedNetwork = "directed_network"
	TopologyDependency      = "dependency"
	TopologyTree            = "tree"
)

// Never ever change these UUIDs!!!
var (
	SystemWorkItemLinkTypeBugBlockerID     = uuid.FromStringOrNil("2CEA3C79-3B79-423B-90F4-1E59174C8F43")
	SystemWorkItemLinkPlannerItemRelatedID = uuid.FromStringOrNil("9B631885-83B1-4ABB-A340-3A9EDE8493FA")
	SystemWorkItemLinkTypeParentChildID    = uuid.FromStringOrNil("25C326A7-6D03-4F5A-B23B-86A9EE4171E9")
)

// returns true if the left hand and right hand side string
// pointers either both point to nil or reference the same
// content; otherwise false is returned.
func strPtrIsNilOrContentIsEqual(l, r *string) bool {
	if l == nil && r != nil {
		return false
	}
	if l != nil && r == nil {
		return false
	}
	if l == nil && r == nil {
		return true
	}
	return *l == *r
}

// WorkItemLinkType represents the type of a work item link as it is stored in the db
type WorkItemLinkType struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Name is the unique name of this work item link type.
	Name string
	// Description is an optional description of the work item link type
	Description *string
	// Version for optimistic concurrency control
	Version  int
	Topology string // Valid values: network, directed_network, dependency, tree

	ForwardName string
	ReverseName string

	LinkCategoryID uuid.UUID `sql:"type:uuid"`

	// Reference to one Space
	SpaceID uuid.UUID `sql:"type:uuid"`
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemLinkType{}
var _ convert.Equaler = (*WorkItemLinkType)(nil)

// Equal returns true if two WorkItemLinkType objects are equal; otherwise false is returned.
func (t WorkItemLinkType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkType)
	if !ok {
		return false
	}
	if !t.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !uuid.Equal(t.ID, other.ID) {
		return false
	}
	if t.Name != other.Name {
		return false
	}
	if t.Version != other.Version {
		return false
	}
	if !strPtrIsNilOrContentIsEqual(t.Description, other.Description) {
		return false
	}
	if t.Topology != other.Topology {
		return false
	}
	if t.ForwardName != other.ForwardName {
		return false
	}
	if t.ReverseName != other.ReverseName {
		return false
	}
	if !uuid.Equal(t.LinkCategoryID, other.LinkCategoryID) {
		return false
	}
	if !uuid.Equal(t.SpaceID, other.SpaceID) {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link type
// cannot be used for the creation of a new work item link type.
func (t *WorkItemLinkType) CheckValidForCreation() error {
	if t.Name == "" {
		return errors.NewBadParameterError("name", t.Name)
	}
	if t.ForwardName == "" {
		return errors.NewBadParameterError("forward_name", t.ForwardName)
	}
	if t.ReverseName == "" {
		return errors.NewBadParameterError("reverse_name", t.ReverseName)
	}
	if err := CheckValidTopology(t.Topology); err != nil {
		return errs.WithStack(err)
	}
	if t.LinkCategoryID == uuid.Nil {
		return errors.NewBadParameterError("link_category_id", t.LinkCategoryID)
	}
	if t.SpaceID == uuid.Nil {
		return errors.NewBadParameterError("space_id", t.SpaceID)
	}
	return nil
}

// TableName implements gorm.tabler
func (t WorkItemLinkType) TableName() string {
	return "work_item_link_types"
}

// CheckValidTopology returns nil if the given topology is valid;
// otherwise a BadParameterError is returned.
func CheckValidTopology(t string) error {
	if t != TopologyNetwork && t != TopologyDirectedNetwork && t != TopologyDependency && t != TopologyTree {
		return errors.NewBadParameterError("topolgy", t).Expected(TopologyNetwork + "|" + TopologyDirectedNetwork + "|" + TopologyDependency + "|" + TopologyTree)
	}
	return nil
}

// GetETagData returns the field values to use to generate the ETag
func (t WorkItemLinkType) GetETagData() []interface{} {
	return []interface{}{t.ID, t.Version}
}

// GetLastModified returns the last modification time
func (t WorkItemLinkType) GetLastModified() time.Time {
	return t.UpdatedAt
}
