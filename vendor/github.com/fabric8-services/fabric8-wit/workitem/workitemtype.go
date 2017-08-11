package workitem

import (
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// String constants for the local work item types.
const (
	// pathSep specifies the symbol used to concatenate WIT names to form a so called "path"
	pathSep = "."

	SystemVersion = "version"

	SystemRemoteItemID        = "system.remote_item_id"
	SystemNumber              = "system.number"
	SystemTitle               = "system.title"
	SystemDescription         = "system.description"
	SystemDescriptionMarkup   = "system.description.markup"
	SystemDescriptionRendered = "system.description.rendered"
	SystemState               = "system.state"
	SystemAssignees           = "system.assignees"
	SystemCreator             = "system.creator"
	SystemCreatedAt           = "system.created_at"
	SystemUpdatedAt           = "system.updated_at"
	SystemOrder               = "system.order"
	SystemIteration           = "system.iteration"
	SystemArea                = "system.area"
	SystemCodebase            = "system.codebase"

	SystemStateOpen       = "open"
	SystemStateNew        = "new"
	SystemStateInProgress = "in progress"
	SystemStateResolved   = "resolved"
	SystemStateClosed     = "closed"
)

// Never ever change these UUIDs!!!
var (
	// base item type with common fields for planner item types like userstory, experience, bug, feature, etc.
	SystemPlannerItem      = uuid.FromStringOrNil("86af5178-9b41-469b-9096-57e5155c3f31") // "planneritem"
	SystemTask             = uuid.FromStringOrNil("bbf35418-04b6-426c-a60b-7f80beb0b624") // "task"
	SystemValueProposition = uuid.FromStringOrNil("3194ab60-855b-4155-9005-9dce4a05f1eb") // "valueproposition"
	SystemFundamental      = uuid.FromStringOrNil("ee7ca005-f81d-4eea-9b9b-1965df0988d0") // "fundamental"
	SystemExperience       = uuid.FromStringOrNil("b9a71831-c803-4f66-8774-4193fffd1311") // "experience"
	SystemFeature          = uuid.FromStringOrNil("0a24d3c2-e0a6-4686-8051-ec0ea1915a28") // "feature"
	SystemScenario         = uuid.FromStringOrNil("71171e90-6d35-498f-a6a7-2083b5267c18") // "scenario"
	SystemBug              = uuid.FromStringOrNil("26787039-b68f-4e28-8814-c2f93be1ef4e") // "bug"
	SystemPapercuts        = uuid.FromStringOrNil("6d603ab4-7c5e-4c5f-bba8-a3ba9d370985") // "papercuts"
)

// WorkItemType represents a work item type as it is stored in the db
type WorkItemType struct {
	gormsupport.Lifecycle
	// ID
	ID uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	// Name is a human readable name of this work item type
	Name string
	// Description is an optional description of the work item type
	Description *string
	// The CSS icon class to render an icon for the WIT
	Icon string
	// Version for optimistic concurrency control
	Version int
	// the IDs of the parents, separated with a dot (".") separator
	Path string
	// definitions of the fields this work item type supports
	Fields FieldDefinitions `sql:"type:jsonb"`
	// Reference to one Space
	SpaceID uuid.UUID `sql:"type:uuid"`
}

// GetTypePathSeparator returns the work item type's path separator "."
func GetTypePathSeparator() string {
	return pathSep
}

// LtreeSafeID returns the ID of the work item type in an postgres ltree safe manner.
// The returned string can be used as an ltree node.
func (wit WorkItemType) LtreeSafeID() string {
	return LtreeSafeID(wit.ID)
}

// LtreeSafeID returns the ID of the work item type in an postgres ltree safe manner
// The returned string can be used as an ltree node.
func LtreeSafeID(witID uuid.UUID) string {
	return strings.Replace(witID.String(), "-", "_", -1)
}

// TableName implements gorm.tabler
func (wit WorkItemType) TableName() string {
	return "work_item_types"
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = WorkItemType{}
var _ convert.Equaler = (*WorkItemType)(nil)

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

// Equal returns true if two WorkItemType objects are equal; otherwise false is returned.
func (wit WorkItemType) Equal(u convert.Equaler) bool {
	other, ok := u.(WorkItemType)
	if !ok {
		return false
	}
	if !uuid.Equal(wit.ID, other.ID) {
		return false
	}
	if !wit.Lifecycle.Equal(other.Lifecycle) {
		return false
	}
	if wit.Version != other.Version {
		return false
	}
	if wit.Name != other.Name {
		return false
	}
	if !strPtrIsNilOrContentIsEqual(wit.Description, other.Description) {
		return false
	}
	if wit.Icon != other.Icon {
		return false
	}
	if wit.Path != other.Path {
		return false
	}
	if len(wit.Fields) != len(other.Fields) {
		return false
	}
	for witKey, witVal := range wit.Fields {
		otherVal, keyFound := other.Fields[witKey]
		if !keyFound {
			return false
		}
		if !witVal.Equal(otherVal) {
			return false
		}
	}
	return wit.SpaceID == other.SpaceID
}

// ConvertWorkItemStorageToModel converts a workItem from the storage/persistence layer into a workItem of the model domain layer
func (wit WorkItemType) ConvertWorkItemStorageToModel(workItem WorkItemStorage) (*WorkItem, error) {
	result := WorkItem{
		ID:                     workItem.ID,
		Number:                 workItem.Number,
		Type:                   workItem.Type,
		Version:                workItem.Version,
		Fields:                 map[string]interface{}{},
		SpaceID:                workItem.SpaceID,
		relationShipsChangedAt: workItem.RelationShipsChangedAt,
	}

	for name, field := range wit.Fields {
		var err error
		if name == SystemCreatedAt {
			continue
		}
		result.Fields[name], err = field.ConvertFromModel(name, workItem.Fields[name])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		result.Fields[SystemOrder] = workItem.ExecutionOrder
	}

	return &result, nil
}

// IsTypeOrSubtypeOf returns true if the work item type with the given type ID,
// is of the same type as the current WIT or of it is a subtype; otherwise false
// is returned.
func (wit WorkItemType) IsTypeOrSubtypeOf(typeID uuid.UUID) bool {
	// Check for complete inclusion (e.g. "bar" is contained in "foo.bar.cake")
	// and for suffix (e.g. ".cake" is the suffix of "foo.bar.cake").
	return uuid.Equal(wit.ID, typeID) || strings.Contains(wit.Path, LtreeSafeID(typeID)+pathSep)
}

// GetETagData returns the field values to use to generate the ETag
func (wit WorkItemType) GetETagData() []interface{} {
	return []interface{}{wit.ID, wit.Version}
}

// GetLastModified returns the last modification time
func (wit WorkItemType) GetLastModified() time.Time {
	return wit.UpdatedAt
}
