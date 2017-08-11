package link

import (
	"time"

	convert "github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkTypeCombination stores the allowed work item types for each link type
type WorkItemLinkTypeCombination struct {
	gormsupport.Lifecycle
	ID           uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // ID
	Version      int       // Version for optimistic concurrency control
	SpaceID      uuid.UUID `sql:"type:uuid"` // Reference to one Space
	LinkTypeID   uuid.UUID `sql:"type:uuid"`
	SourceTypeID uuid.UUID `sql:"type:uuid"`
	TargetTypeID uuid.UUID `sql:"type:uuid"`
}

// Ensure WorkItemLinkTypeCombination implements the Equaler interface
var _ convert.Equaler = WorkItemLinkTypeCombination{}
var _ convert.Equaler = (*WorkItemLinkTypeCombination)(nil)

// Equal returns true if two WorkItemLinkTypeCombination objects are equal; otherwise false is returned.
func (tc WorkItemLinkTypeCombination) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemLinkTypeCombination)
	if !ok {
		return false
	}
	if !tc.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if !uuid.Equal(tc.ID, other.ID) {
		return false
	}
	if tc.Version != other.Version {
		return false
	}
	if !uuid.Equal(tc.SpaceID, other.SpaceID) {
		return false
	}
	if !uuid.Equal(tc.LinkTypeID, other.LinkTypeID) {
		return false
	}
	if !uuid.Equal(tc.SourceTypeID, other.SourceTypeID) {
		return false
	}
	if !uuid.Equal(tc.TargetTypeID, other.TargetTypeID) {
		return false
	}
	return true
}

// CheckValidForCreation returns an error if the work item link WorkItemLinkTypeCombination cannot be
// used for the creation of a new work item link WorkItemLinkTypeCombination.
func (tc *WorkItemLinkTypeCombination) CheckValidForCreation() error {
	if uuid.Equal(tc.SpaceID, uuid.Nil) {
		return errors.NewBadParameterError("space_id", tc.SpaceID)
	}
	if uuid.Equal(tc.LinkTypeID, uuid.Nil) {
		return errors.NewBadParameterError("link_type_id", tc.LinkTypeID)
	}
	if uuid.Equal(tc.SourceTypeID, uuid.Nil) {
		return errors.NewBadParameterError("source_type", tc.SourceTypeID)
	}
	if uuid.Equal(tc.TargetTypeID, uuid.Nil) {
		return errors.NewBadParameterError("target_type", tc.TargetTypeID)
	}
	return nil
}

// TableName implements gorm.tabler
func (tc WorkItemLinkTypeCombination) TableName() string {
	return "work_item_link_type_combinations"
}

// GetETagData returns the field values to use to generate the ETag
func (tc WorkItemLinkTypeCombination) GetETagData() []interface{} {
	return []interface{}{tc.ID, tc.Version}
}

// GetLastModified returns the last modification time
func (tc WorkItemLinkTypeCombination) GetLastModified() time.Time {
	return tc.UpdatedAt
}
