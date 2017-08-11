package link

import (
	"context"

	"time"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// RevisionRepository encapsulates storage & retrieval of historical versions of work item links
type RevisionRepository interface {
	// Create stores a new revision for the given work item link.
	Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, l WorkItemLink) error
	// List retrieves all revisions for a given work item link
	List(ctx context.Context, workitemID uuid.UUID) ([]Revision, error)
}

// NewRevisionRepository creates a GormCommentRevisionRepository
func NewRevisionRepository(db *gorm.DB) *GormWorkItemLinkRevisionRepository {
	repository := &GormWorkItemLinkRevisionRepository{db}
	return repository
}

// GormCommentRevisionRepository implements CommentRevisionRepository using gorm
type GormWorkItemLinkRevisionRepository struct {
	db *gorm.DB
}

// Create stores a new revision for the given work item link.
func (r *GormWorkItemLinkRevisionRepository) Create(ctx context.Context, modifierID uuid.UUID, revisionType RevisionType, l WorkItemLink) error {
	log.Info(nil, map[string]interface{}{
		"modifier_id":   modifierID,
		"revision_type": revisionType,
	}, "Storing a revision after operation on work item link.")
	tx := r.db
	revision := &Revision{
		ModifierIdentity:     modifierID,
		Time:                 time.Now(),
		Type:                 revisionType,
		WorkItemLinkID:       l.ID,
		WorkItemLinkVersion:  l.Version,
		WorkItemLinkSourceID: l.SourceID,
		WorkItemLinkTargetID: l.TargetID,
		WorkItemLinkTypeID:   l.LinkTypeID,
	}
	if err := tx.Create(&revision).Error; err != nil {
		return errors.NewInternalError(ctx, errs.Wrap(err, "failed to create new work item link revision"))
	}
	log.Debug(ctx, map[string]interface{}{"wil_id": l.ID}, "work item link revision occurrence created")
	return nil
}

// List retrieves all revisions for a given work item link
func (r *GormWorkItemLinkRevisionRepository) List(ctx context.Context, linkID uuid.UUID) ([]Revision, error) {
	log.Debug(nil, map[string]interface{}{}, "List all revisions for work item link with ID=%v", linkID.String())
	var revisions []Revision
	if err := r.db.Where("work_item_link_id = ?", linkID.String()).Order("revision_time asc").Find(&revisions).Error; err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "failed to retrieve work item link revisions"))
	}
	return revisions, nil
}
