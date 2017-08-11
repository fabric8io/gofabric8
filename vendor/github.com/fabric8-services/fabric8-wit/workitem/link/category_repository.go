package link

import (
	"context"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategoryRepository encapsulates storage & retrieval of work item link categories
type WorkItemLinkCategoryRepository interface {
	repository.Exister
	Create(ctx context.Context, linkCat *WorkItemLinkCategory) (*WorkItemLinkCategory, error)
	Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkCategory, error)
	List(ctx context.Context) ([]WorkItemLinkCategory, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Save(ctx context.Context, linkCat WorkItemLinkCategory) (*WorkItemLinkCategory, error)
}

// NewWorkItemLinkCategoryRepository creates a work item link category repository based on gorm
func NewWorkItemLinkCategoryRepository(db *gorm.DB) *GormWorkItemLinkCategoryRepository {
	return &GormWorkItemLinkCategoryRepository{db}
}

// GormWorkItemLinkCategoryRepository implements WorkItemLinkCategoryRepository using gorm
type GormWorkItemLinkCategoryRepository struct {
	db *gorm.DB
}

// Create creates a new work item link category in the repository.
// Returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Create(ctx context.Context, linkCat *WorkItemLinkCategory) (*WorkItemLinkCategory, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "create"}, time.Now())
	if linkCat.Name == "" {
		return nil, errors.NewBadParameterError("name", linkCat.Name)
	}
	db := r.db.Create(linkCat)
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	log.Info(ctx, map[string]interface{}{
		"wilc_id": linkCat.ID,
	}, "work item link category created")
	return linkCat, nil
}

// Load returns the work item link category for the given ID.
// Returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Load(ctx context.Context, ID uuid.UUID) (*WorkItemLinkCategory, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "load"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wilc_id": ID,
	}, "Loading work item link category")
	result := WorkItemLinkCategory{}
	db := r.db.Model(&result).Where("id=?", ID).First(&result)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": ID,
		}, "work item link category not found by id ", ID)
		return nil, errors.NewNotFoundError("work item link category", ID.String())
	}
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	return &result, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemLinkCategoryRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "exists"}, time.Now())
	return repository.CheckExists(ctx, r.db, WorkItemLinkCategory{}.TableName(), id)
}

// List returns all work item link categories
// TODO: Handle pagination
func (r *GormWorkItemLinkCategoryRepository) List(ctx context.Context) ([]WorkItemLinkCategory, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "list"}, time.Now())
	var rows []WorkItemLinkCategory
	db := r.db.Find(&rows)
	if db.Error != nil {
		return nil, db.Error
	}
	return rows, nil
}

// Delete deletes the work item link category with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "delete"}, time.Now())
	var cat = WorkItemLinkCategory{
		ID: ID,
	}
	log.Info(ctx, map[string]interface{}{
		"wilc_id": ID,
	}, "Work item link category to delete")
	db := r.db.Delete(&cat)
	if db.Error != nil {
		return errors.NewInternalError(ctx, db.Error)
	}
	if db.RowsAffected == 0 {
		return errors.NewNotFoundError("work item link category", ID.String())
	}
	return nil
}

// Save updates the given work item link category in storage. Version must be the same as the one int the stored version.
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemLinkCategoryRepository) Save(ctx context.Context, linkCat WorkItemLinkCategory) (*WorkItemLinkCategory, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemlinkcategory", "save"}, time.Now())
	res := WorkItemLinkCategory{}

	db := r.db.Model(&res).Where("id=?", linkCat.ID).First(&res)
	if db.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": linkCat.ID,
		}, "work item link category not found")
		return nil, errors.NewNotFoundError("work item link category", linkCat.ID.String())
	}
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": linkCat.ID,
			"err":     db.Error,
		}, "unable to find work item link category")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	if res.Version != linkCat.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	newLinkCat := WorkItemLinkCategory{
		ID:          linkCat.ID,
		Version:     linkCat.Version + 1,
		Name:        linkCat.Name,
		Description: linkCat.Description,
	}
	db = db.Save(&newLinkCat)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"wilc_id": newLinkCat.ID,
			"err":     db.Error,
		}, "unable to save work item link category repository")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	log.Info(ctx, map[string]interface{}{
		"wilc_id":         newLinkCat.ID,
		"newLinkCategory": newLinkCat,
	}, "Work item link category updated")
	return &newLinkCat, nil
}
