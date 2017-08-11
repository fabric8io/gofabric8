package remoteworkitem

import (
	"github.com/fabric8-services/fabric8-wit/gormsupport"

	uuid "github.com/satori/go.uuid"
)

// TrackerQuery represents tracker query
type TrackerQuery struct {
	gormsupport.Lifecycle
	ID uint64 `gorm:"primary_key"`
	// Search query of the tracker
	Query string
	// Schedule to fetch and import remote tracker items
	Schedule string
	// TrackerID is a foreign key for a tracker
	TrackerID uint64 `gorm:"ForeignKey:Tracker"`
	// SpaceID is a foreign key for a space
	SpaceID uuid.UUID `gorm:"ForeignKey:Space"`
}
