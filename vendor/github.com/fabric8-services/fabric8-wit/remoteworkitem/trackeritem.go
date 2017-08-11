package remoteworkitem

import "github.com/fabric8-services/fabric8-wit/gormsupport"

// TrackerItem represents a remote tracker item
// Staging area before pushing to work item
type TrackerItem struct {
	gormsupport.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// Remote item ID - unique across multiple trackers
	RemoteItemID string `gorm:"not null;unique"`
	// the field values
	Item string
	// FK to tracker
	TrackerID uint64 `gorm:"ForeignKey:Tracker"`
}
