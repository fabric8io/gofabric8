package workitem

import (
	"context"

	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// RevisionRepository encapsulates storage & retrieval of historical versions of work items
type RevisionRepository interface {
	// Create stores a new revision for the given work item.
	Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, workitem WorkItemStorage) error
	// List retrieves all revisions for a given work item
	List(ctx context.Context, workitemID uuid.UUID) ([]Revision, error)
}

// NewRevisionRepository creates a GormRevisionRepository
func NewRevisionRepository(db *gorm.DB) *GormRevisionRepository {
	repository := &GormRevisionRepository{db}
	return repository
}

// GormRevisionRepository implements RevisionRepository using gorm
type GormRevisionRepository struct {
	db *gorm.DB
}

// Create stores a new revision for the given work item.
func (r *GormRevisionRepository) Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, workitem WorkItemStorage) error {
	log.Info(nil, map[string]interface{}{
		"modifier_id":   modifierID,
		"revision_type": revisionType,
	}, "Storing a revision after operation on work item.")
	tx := r.db
	workitemRevision := &Revision{
		ModifierIdentity: modifierID,
		Time:             time.Now(),
		Type:             revisionType,
		WorkItemID:       workitem.ID,
		WorkItemTypeID:   workitem.Type,
		WorkItemVersion:  workitem.Version,
		WorkItemFields:   workitem.Fields,
	}
	// do not store fields when the work item is deleted
	if workitemRevision.Type == RevisionTypeDelete {
		workitemRevision.WorkItemFields = Fields{}
	}
	if err := tx.Create(&workitemRevision).Error; err != nil {
		return errors.NewInternalError(ctx, errs.Wrap(err, "failed to create new work item revision"))
	}
	log.Debug(ctx, map[string]interface{}{"wi_id": workitem.ID}, "Work item revision occurrence created")
	return nil
}

// List retrieves all revisions for a given work item
func (r *GormRevisionRepository) List(ctx context.Context, workitemID uuid.UUID) ([]Revision, error) {
	log.Debug(nil, map[string]interface{}{}, "List all revisions for work item with ID=%v", workitemID)
	var revisions []Revision
	if err := r.db.Where("work_item_id = ?", workitemID).Order("revision_time asc").Find(&revisions).Error; err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "failed to retrieve work item revisions"))
	}
	return revisions, nil
}
