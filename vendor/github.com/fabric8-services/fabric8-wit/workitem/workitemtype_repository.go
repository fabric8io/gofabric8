package workitem

import (
	"time"

	"context"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/path"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var cache = NewWorkItemTypeCache()

// WorkItemTypeRepository encapsulates storage & retrieval of work item types
type WorkItemTypeRepository interface {
	repository.Exister
	Load(ctx context.Context, spaceID uuid.UUID, id uuid.UUID) (*WorkItemType, error)
	Create(ctx context.Context, spaceID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields map[string]FieldDefinition) (*WorkItemType, error)
	CreateFromModel(ctx context.Context, model *WorkItemType) (*WorkItemType, error)
	List(ctx context.Context, spaceID uuid.UUID, start *int, length *int) ([]WorkItemType, error)
	ListPlannerItems(ctx context.Context, spaceID uuid.UUID) ([]WorkItemType, error)
}

// NewWorkItemTypeRepository creates a wi type repository based on gorm
func NewWorkItemTypeRepository(db *gorm.DB) *GormWorkItemTypeRepository {
	return &GormWorkItemTypeRepository{db}
}

// GormWorkItemTypeRepository implements WorkItemTypeRepository using gorm
type GormWorkItemTypeRepository struct {
	db *gorm.DB
}

// LoadByID returns the work item for the given id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) LoadByID(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	res, err := r.LoadTypeFromDB(ctx, id)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	return res, nil
}

// Load returns the work item for the given spaceID and id
// returns NotFoundError, InternalError
func (r *GormWorkItemTypeRepository) Load(ctx context.Context, spaceID uuid.UUID, id uuid.UUID) (*WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "load"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wit_id":   id,
		"space_id": spaceID,
	}, "Loading work item type")
	res, ok := cache.Get(id)
	if !ok {
		log.Info(ctx, map[string]interface{}{
			"wit_id":   id,
			"space_id": spaceID,
		}, "Work item type doesn't exist in the cache. Loading from DB...")
		res = WorkItemType{}

		db := r.db.Model(&res).Where("id=? AND space_id=?", id, spaceID).First(&res)
		if db.RecordNotFound() {
			log.Error(ctx, map[string]interface{}{
				"wit_id":   id,
				"space_id": spaceID,
			}, "work item type not found")
			return nil, errors.NewNotFoundError("work item type", id.String())
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		cache.Put(res)
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemTypeRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "exists"}, time.Now())
	log.Info(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Checking if work item type exists")

	uuid, err := uuid.FromString(id)
	if err != nil {
		return errors.NewBadParameterError("id", id)
	}
	_, exists := cache.Get(uuid)
	if exists {
		return nil
	}
	return repository.CheckExists(ctx, r.db, WorkItemType{}.TableName(), id)
}

// LoadTypeFromDB return work item type for the given id
func (r *GormWorkItemTypeRepository) LoadTypeFromDB(ctx context.Context, id uuid.UUID) (*WorkItemType, error) {
	log.Info(ctx, map[string]interface{}{
		"wit_id": id,
	}, "Loading work item type")
	res, ok := cache.Get(id)
	if !ok {
		log.Info(ctx, map[string]interface{}{
			"wit_id": id,
		}, "Work item type doesn't exist in the cache. Loading from DB...")
		res = WorkItemType{}
		db := r.db.Model(&res).Where("id=?", id).First(&res)
		if db.RecordNotFound() {
			log.Error(ctx, map[string]interface{}{
				"wit_id": id,
			}, "work item type not found")
			return nil, errors.NewNotFoundError("work item type", id.String())
		}
		if err := db.Error; err != nil {
			log.Error(ctx, map[string]interface{}{
				"witID": id,
			}, "work item type retrieval error", err.Error())
			return nil, errors.NewInternalError(ctx, err)
		}
		cache.Put(res)
	}
	return &res, nil
}

// ClearGlobalWorkItemTypeCache removes all work items from the global cache
func ClearGlobalWorkItemTypeCache() {
	cache.Clear()
}

// CreateFromModel creates a new work item type in the repository without any
// fancy stuff.
func (r *GormWorkItemTypeRepository) CreateFromModel(ctx context.Context, model *WorkItemType) (*WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "createfrommodel"}, time.Now())
	// Make sure this WIT has an ID
	if model.ID == uuid.Nil {
		model.ID = uuid.NewV4()
	}

	if err := r.db.Create(&model).Error; err != nil {
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "failed to create work item type"))
	}

	log.Debug(ctx, map[string]interface{}{"witID": model.ID}, "work item type created successfully!")
	return model, nil
}

// Create creates a new work item type in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemTypeRepository) Create(ctx context.Context, spaceID uuid.UUID, id *uuid.UUID, extendedTypeID *uuid.UUID, name string, description *string, icon string, fields map[string]FieldDefinition) (*WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "create"}, time.Now())
	// Make sure this WIT has an ID
	if id == nil {
		tmpID := uuid.NewV4()
		id = &tmpID
	}

	allFields := map[string]FieldDefinition{}
	path := LtreeSafeID(*id)
	if extendedTypeID != nil {
		extendedType := WorkItemType{}
		db := r.db.Model(&extendedType).Where("id=?", extendedTypeID).First(&extendedType)
		if db.RecordNotFound() {
			return nil, errors.NewBadParameterError("extendedTypeID", *extendedTypeID)
		}
		if err := db.Error; err != nil {
			return nil, errors.NewInternalError(ctx, err)
		}
		// copy fields from extended type
		for key, value := range extendedType.Fields {
			allFields[key] = value
		}
		path = extendedType.Path + pathSep + path
	}
	// now process new fields, checking whether they are already there.
	for field, definition := range fields {
		existing, exists := allFields[field]
		if exists && !compatibleFields(existing, definition) {
			return nil, errs.Errorf("incompatible change for field %s", field)
		}
		allFields[field] = definition
	}

	model := WorkItemType{
		Version:     0,
		ID:          *id,
		Name:        name,
		Description: description,
		Icon:        icon,
		Path:        path,
		Fields:      allFields,
		SpaceID:     spaceID,
	}

	return r.CreateFromModel(ctx, &model)
}

// List returns work item types that derives from PlannerItem type
func (r *GormWorkItemTypeRepository) ListPlannerItems(ctx context.Context, spaceID uuid.UUID) ([]WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "listPlannerItems"}, time.Now())
	var rows []WorkItemType
	path := path.Path{}
	db := r.db.Select("id").Where("space_id = ? AND path::text LIKE '"+path.ConvertToLtree(SystemPlannerItem)+".%'", spaceID.String())

	if err := db.Find(&rows).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      err,
		}, "unable to list the work item types that derive of planner item")
		return nil, errs.WithStack(err)
	}
	return rows, nil
}

// List returns work item types selected by the given criteria.Expression,
// starting with start (zero-based) and returning at most "limit" item types.
func (r *GormWorkItemTypeRepository) List(ctx context.Context, spaceID uuid.UUID, start *int, limit *int) ([]WorkItemType, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitemtype", "list"}, time.Now())
	// Currently we don't implement filtering here, so leave this empty
	// TODO: (kwk) implement criteria parsing just like for work items
	var rows []WorkItemType
	db := r.db.Where("space_id = ?", spaceID)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, errs.WithStack(err)
	}
	return rows, nil
}
