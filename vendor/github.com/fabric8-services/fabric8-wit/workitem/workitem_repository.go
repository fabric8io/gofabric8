package workitem

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/space"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const orderValue = 1000

type DirectionType string

const (
	DirectionAbove  DirectionType = "above"
	DirectionBelow  DirectionType = "below"
	DirectionTop    DirectionType = "top"
	DirectionBottom DirectionType = "bottom"
)

// WorkItemRepository encapsulates storage & retrieval of work items
type WorkItemRepository interface {
	repository.Exister
	Load(ctx context.Context, spaceID uuid.UUID, wiNumber int) (*WorkItem, error)
	LoadByID(ctx context.Context, id uuid.UUID) (*WorkItem, error)
	LookupIDByNamedSpaceAndNumber(ctx context.Context, ownerName, spaceName string, wiNumber int) (*uuid.UUID, *uuid.UUID, error)
	Save(ctx context.Context, spaceID uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error)
	Reorder(ctx context.Context, spaceID uuid.UUID, direction DirectionType, targetID *uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error)
	Delete(ctx context.Context, id uuid.UUID, suppressorID uuid.UUID) error
	Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*WorkItem, error)
	List(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, length *int) ([]WorkItem, int, error)
	Fetch(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (*WorkItem, error)
	GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error)
	GetCountsForIteration(ctx context.Context, itr *iteration.Iteration) (map[string]WICountsPerIteration, error)
	Count(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (int, error)
}

// NewWorkItemRepository creates a GormWorkItemRepository
func NewWorkItemRepository(db *gorm.DB) *GormWorkItemRepository {
	repository := &GormWorkItemRepository{db, &GormWorkItemTypeRepository{db}, &GormRevisionRepository{db}}
	return repository
}

// GormWorkItemRepository implements WorkItemRepository using gorm
type GormWorkItemRepository struct {
	db   *gorm.DB
	witr *GormWorkItemTypeRepository
	wirr *GormRevisionRepository
}

// ************************************************
// WorkItemRepository implementation
// ************************************************

// LoadFromDB returns the work item with the given natural ID in model representation.
func (r *GormWorkItemRepository) LoadFromDB(ctx context.Context, id uuid.UUID) (*WorkItemStorage, error) {
	log.Info(nil, map[string]interface{}{
		"wi_id": id,
	}, "Loading work item")

	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id = ?", id).First(&res)
	if tx.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wi_id": id,
		}, "work item not found")
		return nil, errors.NewNotFoundError("work item", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res, nil
}

// LoadByID returns the work item for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadByID(ctx context.Context, id uuid.UUID) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "loadById"}, time.Now())
	res, err := r.LoadFromDB(ctx, id)
	if err != nil {
		return nil, errs.WithStack(err)
	}
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	return ConvertWorkItemStorageToModel(wiType, res)
}

// Load returns the work item for the given spaceID and item id
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) Load(ctx context.Context, spaceID uuid.UUID, wiNumber int) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "load"}, time.Now())
	wiStorage, wiType, err := r.loadWorkItemStorage(ctx, spaceID, wiNumber, false)
	if err != nil {
		return nil, err
	}
	return ConvertWorkItemStorageToModel(wiType, wiStorage)
}

// LookupIDByNamedSpaceAndNumber returns the work item's ID for the given owner name, space name and item number
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LookupIDByNamedSpaceAndNumber(ctx context.Context, ownerName, spaceName string, wiNumber int) (*uuid.UUID, *uuid.UUID, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "lookupIDByNamedSpaceAndNumber"}, time.Now())
	log.Debug(nil, map[string]interface{}{
		"wi_number":  wiNumber,
		"space_name": spaceName,
		"owner_name": ownerName,
	}, "Loading work item")
	query := fmt.Sprintf("select wi.id, wi.space_id from %[1]s wi "+
		"join %[2]s s on wi.space_id = s.id "+
		"join %[3]s i on s.owner_id = i.id "+
		"where lower(i.username) = lower(?) and lower(s.name) = lower(?) and wi.number = ?",
		WorkItemStorage{}.TableName(), space.Space{}.TableName(), account.Identity{}.TableName())
	// 'scan' destination must be slice or struct
	type Result struct {
		WiID uuid.UUID `gorm:"column:id"`
		// TODO(xcoulon) SpaceID can be removed once PR for #1452 is merged, as we won't need it anymore in the controller
		SpaceID uuid.UUID
	}
	var result Result
	db := r.db.Raw(query, ownerName, spaceName, wiNumber).Scan(&result)
	if db.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wi_number":  wiNumber,
			"space_name": spaceName,
			"owner_name": ownerName,
		}, "work item not found")
		return nil, nil, errors.NewNotFoundError("work item", strconv.Itoa(wiNumber))
	}
	if db.Error != nil {
		return nil, nil, errors.NewInternalError(ctx, errs.Wrap(db.Error, "error while looking up a work item ID"))
	}
	log.Debug(ctx, map[string]interface{}{
		"wi_number":  wiNumber,
		"space_name": spaceName,
		"owner_name": ownerName,
	}, "Matching work item with ID='%s' in space with ID='%s'", result.WiID.String(), result.SpaceID.String())
	return &result.WiID, &result.SpaceID, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormWorkItemRepository) CheckExists(ctx context.Context, workitemID string) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "exists"}, time.Now())
	return repository.CheckExists(ctx, r.db, workitemTableName, workitemID)
}

func (r *GormWorkItemRepository) loadWorkItemStorage(ctx context.Context, spaceID uuid.UUID, wiNumber int, selectForUpdate bool) (*WorkItemStorage, *WorkItemType, error) {
	log.Debug(nil, map[string]interface{}{
		"wi_number": wiNumber,
		"space_id":  spaceID,
	}, "Loading work item")
	wiStorage := &WorkItemStorage{}
	// SELECT ... FOR UPDATE will lock the row to prevent concurrent update while until surrounding transaction ends.
	tx := r.db
	if selectForUpdate {
		tx = tx.Set("gorm:query_option", "FOR UPDATE")
	}
	tx = tx.Model(wiStorage).Where("number=? AND space_id=?", wiNumber, spaceID).First(wiStorage)
	if tx.RecordNotFound() {
		log.Error(nil, map[string]interface{}{
			"wi_number": wiNumber,
			"space_id":  spaceID,
		}, "work item not found")
		return nil, nil, errors.NewNotFoundError("work item", strconv.Itoa(wiNumber))
	}
	if tx.Error != nil {
		return nil, nil, errors.NewInternalError(ctx, tx.Error)
	}
	wiType, err := r.witr.LoadTypeFromDB(ctx, wiStorage.Type)
	if err != nil {
		return nil, nil, errors.NewInternalError(ctx, err)
	}
	return wiStorage, wiType, nil
}

// LoadTopWorkitem returns top most work item of the list. Top most workitem has the Highest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadTopWorkitem(ctx context.Context, spaceID uuid.UUID) (*WorkItem, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s where space_id=?)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query, spaceID).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// LoadBottomWorkitem returns bottom work item of the list. Bottom most workitem has the lowest order.
// returns NotFoundError, ConversionError or InternalError
func (r *GormWorkItemRepository) LoadBottomWorkitem(ctx context.Context, spaceID uuid.UUID) (*WorkItem, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT min(execution_order) FROM %[1]s where space_id=?)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query, spaceID).First(&res)
	wiType, err := r.witr.LoadTypeFromDB(ctx, res.Type)
	if err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// LoadHighestOrder returns the highest execution order in the given space
func (r *GormWorkItemRepository) LoadHighestOrder(ctx context.Context, spaceID uuid.UUID) (float64, error) {
	res := WorkItemStorage{}
	db := r.db.Model(WorkItemStorage{})
	query := fmt.Sprintf("execution_order = (SELECT max(execution_order) FROM %[1]s where space_id=?)",
		WorkItemStorage{}.TableName(),
	)
	db = db.Where(query, spaceID).First(&res)
	order, err := strconv.ParseFloat(fmt.Sprintf("%v", res.ExecutionOrder), 64)
	if err != nil {
		return 0, errors.NewInternalError(ctx, err)
	}
	return order, nil
}

// Delete deletes the work item with the given id
// returns NotFoundError or InternalError
func (r *GormWorkItemRepository) Delete(ctx context.Context, workitemID uuid.UUID, suppressorID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "delete"}, time.Now())
	var workItem = WorkItemStorage{}
	workItem.ID = workitemID
	// retrieve the current version of the work item to delete
	r.db.Select("id, version, type").Where("id = ?", workitemID).Find(&workItem)
	// delete the work item
	tx := r.db.Delete(workItem)
	if err := tx.Error; err != nil {
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		return errors.NewNotFoundError("work item", workitemID.String())
	}
	// store a revision of the deleted work item
	if err := r.wirr.Create(context.Background(), suppressorID, RevisionTypeDelete, workItem); err != nil {
		return errs.Wrapf(err, "error while deleting work item")
	}
	log.Debug(ctx, map[string]interface{}{"wi_id": workitemID}, "Work item deleted successfully!")
	return nil
}

// CalculateOrder calculates the order of the reorder workitem
func (r *GormWorkItemRepository) CalculateOrder(above, below *float64) float64 {
	return (*above + *below) / 2
}

// FindSecondItem returns the order of the second workitem required to reorder.
// Reordering a workitem requires order of two closest workitems: above and below.
// If direction == "above", then
//	FindFirstItem returns the value above which reorder item has to be placed
//      FindSecondItem returns the value below which reorder item has to be placed
// If direction == "below", then
//	FindFirstItem returns the value below which reorder item has to be placed
//      FindSecondItem returns the value above which reorder item has to be placed
func (r *GormWorkItemRepository) FindSecondItem(ctx context.Context, order *float64, spaceID uuid.UUID, secondItemDirection DirectionType) (*uuid.UUID, *float64, error) {
	Item := WorkItemStorage{}
	var tx *gorm.DB
	switch secondItemDirection {
	case DirectionAbove:
		// Finds the item above which reorder item has to be placed
		query := fmt.Sprintf(`execution_order = (SELECT max(execution_order) FROM %[1]s WHERE space_id=? AND (execution_order < ?))`, WorkItemStorage{}.TableName())
		tx = r.db.Where(query, spaceID, order).First(&Item)
	case DirectionBelow:
		// Finds the item below which reorder item has to be placed
		query := fmt.Sprintf("execution_order = (SELECT min(execution_order) FROM %[1]s WHERE space_id=? AND (execution_order > ?))", WorkItemStorage{}.TableName())
		tx = r.db.Where(query, spaceID, order).First(&Item)
	default:
		return nil, nil, nil
	}
	if tx.RecordNotFound() {
		// Item is placed at first or last position
		ItemID := Item.ID
		return &ItemID, nil, nil
	}
	if tx.Error != nil {
		return nil, nil, errors.NewInternalError(ctx, tx.Error)
	}
	ItemID := Item.ID
	return &ItemID, &Item.ExecutionOrder, nil
}

// FindFirstItem returns the order of the target workitem
func (r *GormWorkItemRepository) FindFirstItem(ctx context.Context, spaceID uuid.UUID, id uuid.UUID) (*float64, error) {
	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id=? and space_id=?", id, spaceID).First(&res)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res.ExecutionOrder, nil
}

// Reorder places the to-be-reordered workitem above the input workitem.
// The order of workitems are spaced by a factor of 1000.
// The new order of workitem := (order of previousitem + order of nextitem)/2
// Version must be the same as the one int the stored version
func (r *GormWorkItemRepository) Reorder(ctx context.Context, spaceID uuid.UUID, direction DirectionType, targetID *uuid.UUID, wi WorkItem, modifierID uuid.UUID) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "reorder"}, time.Now())
	var order float64
	res := WorkItemStorage{}
	tx := r.db.Model(WorkItemStorage{}).Where("id = ?", wi.ID).First(&res)
	if tx.RecordNotFound() {
		return nil, errors.NewNotFoundError("work item", wi.ID.String())
	}
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	if res.Version != wi.Version {
		log.Info(ctx, map[string]interface{}{
			"wi_id":           wi.ID.String(),
			"current_version": res.Version,
			"input_version":   wi.Version},
			"version_conflict while reordering items")
		return nil, errors.NewVersionConflictError("version conflict")
	}

	wiType, err := r.witr.LoadTypeFromDB(ctx, wi.Type)
	if err != nil {
		return nil, errors.NewBadParameterError("Type", wi.Type)
	}

	switch direction {
	case DirectionBelow:
		// if direction == "below", place the reorder item **below** the workitem having id equal to targetID
		aboveItemOrder, err := r.FindFirstItem(ctx, spaceID, *targetID)
		if aboveItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		belowItemID, belowItemOrder, err := r.FindSecondItem(ctx, aboveItemOrder, spaceID, DirectionAbove)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		if belowItemOrder == nil {
			// Item is placed at last position
			belowItemOrder := float64(0)
			order = r.CalculateOrder(aboveItemOrder, &belowItemOrder)
		} else if *belowItemID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionAbove:
		// if direction == "above", place the reorder item **above** the workitem having id equal to targetID
		belowItemOrder, err := r.FindFirstItem(ctx, spaceID, *targetID)
		if belowItemOrder == nil || err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		aboveItemID, aboveItemOrder, err := r.FindSecondItem(ctx, belowItemOrder, spaceID, DirectionBelow)
		if err != nil {
			return nil, errors.NewNotFoundError("work item", targetID.String())
		}
		if aboveItemOrder == nil {
			// Item is placed at first position
			order = *belowItemOrder + float64(orderValue)
		} else if *aboveItemID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			order = r.CalculateOrder(aboveItemOrder, belowItemOrder)
		}
	case DirectionTop:
		// if direction == "top", place the reorder item at the topmost position. Now, the reorder item has the highest order in the whole list.
		res, err := r.LoadTopWorkitem(ctx, spaceID)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			topItemOrder := res.Fields[SystemOrder].(float64)
			order = topItemOrder + orderValue
		}
	case DirectionBottom:
		// if direction == "bottom", place the reorder item at the bottom most position. Now, the reorder item has the lowest order in the whole list
		res, err := r.LoadBottomWorkitem(ctx, spaceID)
		if err != nil {
			return nil, errs.Wrapf(err, "Failed to reorder")
		}
		if wi.ID == res.ID {
			// When same reorder request is made again
			order = wi.Fields[SystemOrder].(float64)
		} else {
			bottomItemOrder := res.Fields[SystemOrder].(float64)
			order = bottomItemOrder / 2
		}
	default:
		return &wi, nil
	}
	res.Version = res.Version + 1
	res.Type = wi.Type
	res.Fields = Fields{}

	res.ExecutionOrder = order

	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := wi.Fields[fieldName]
		var err error
		res.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}
	tx = tx.Where("Version = ?", wi.Version).Save(&res)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, res)
	if err != nil {
		return nil, err
	}
	return ConvertWorkItemStorageToModel(wiType, &res)
}

// Save updates the given work item in storage. Version must be the same as the one int the stored version
// returns NotFoundError, VersionConflictError, ConversionError or InternalError
func (r *GormWorkItemRepository) Save(ctx context.Context, spaceID uuid.UUID, updatedWorkItem WorkItem, modifierID uuid.UUID) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "save"}, time.Now())
	wiStorage, wiType, err := r.loadWorkItemStorage(ctx, spaceID, updatedWorkItem.Number, true)
	if err != nil {
		return nil, err
	}
	if wiStorage.Version != updatedWorkItem.Version {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	wiStorage.Version = wiStorage.Version + 1
	wiStorage.Type = updatedWorkItem.Type
	wiStorage.Fields = Fields{}
	wiStorage.ExecutionOrder = updatedWorkItem.Fields[SystemOrder].(float64)
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := updatedWorkItem.Fields[fieldName]
		var err error
		wiStorage.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
	}
	tx := r.db.Where("Version = ?", updatedWorkItem.Version).Save(&wiStorage)
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"wi_id":    updatedWorkItem.ID,
			"space_id": spaceID,
			"version":  updatedWorkItem.Version,
			"err":      err,
		}, "unable to save new version of the work item")
		return nil, errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}
	// store a revision of the modified work item
	err = r.wirr.Create(context.Background(), modifierID, RevisionTypeUpdate, *wiStorage)
	if err != nil {
		return nil, errs.Wrapf(err, "error while saving work item")
	}
	log.Info(ctx, map[string]interface{}{
		"wi_id":    updatedWorkItem.ID,
		"space_id": spaceID,
	}, "Updated work item repository")
	return ConvertWorkItemStorageToModel(wiType, wiStorage)
}

// Create creates a new work item in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormWorkItemRepository) Create(ctx context.Context, spaceID uuid.UUID, typeID uuid.UUID, fields map[string]interface{}, creatorID uuid.UUID) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "create"}, time.Now())

	wiType, err := r.witr.LoadTypeFromDB(ctx, typeID)
	if err != nil {
		return nil, errors.NewBadParameterError("typeID", typeID)
	}
	// retrieve the current issue number in the given space
	numberSequence := WorkItemNumberSequence{}
	tx := r.db.Model(&WorkItemNumberSequence{}).Set("gorm:query_option", "FOR UPDATE").Where("space_id = ?", spaceID).First(&numberSequence)
	if tx.RecordNotFound() {
		numberSequence.SpaceID = spaceID
		numberSequence.CurrentVal = 1
	} else {
		numberSequence.CurrentVal++
	}
	if err = r.db.Save(&numberSequence).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to create work item")
	}

	// The order of workitems are spaced by a factor of 1000.
	pos, err := r.LoadHighestOrder(ctx, spaceID)
	if err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}
	pos = pos + orderValue
	wi := WorkItemStorage{
		Type:           typeID,
		Fields:         Fields{},
		ExecutionOrder: pos,
		SpaceID:        spaceID,
		Number:         numberSequence.CurrentVal,
	}
	fields[SystemCreator] = creatorID.String()
	for fieldName, fieldDef := range wiType.Fields {
		if fieldName == SystemCreatedAt || fieldName == SystemUpdatedAt || fieldName == SystemOrder {
			continue
		}
		fieldValue := fields[fieldName]
		var err error
		wi.Fields[fieldName], err = fieldDef.ConvertToModel(fieldName, fieldValue)
		if err != nil {
			return nil, errors.NewBadParameterError(fieldName, fieldValue)
		}
		if fieldName == SystemDescription && wi.Fields[fieldName] != nil {
			description := rendering.NewMarkupContentFromMap(wi.Fields[fieldName].(map[string]interface{}))
			if !rendering.IsMarkupSupported(description.Markup) {
				return nil, errors.NewBadParameterError(fieldName, fieldValue)
			}
		}
	}
	if err = r.db.Create(&wi).Error; err != nil {
		return nil, errs.Wrapf(err, "failed to create work item")
	}

	witem, err := ConvertWorkItemStorageToModel(wiType, &wi)
	if err != nil {
		return nil, err
	}
	// store a revision of the created work item
	err = r.wirr.Create(context.Background(), creatorID, RevisionTypeCreate, wi)
	if err != nil {
		return nil, errs.Wrapf(err, "error while creating work item")
	}
	log.Debug(ctx, map[string]interface{}{"pkg": "workitem", "wi_id": wi.ID}, "Work item created successfully!")
	return witem, nil
}

// ConvertWorkItemStorageToModel convert work item model to app WI
func ConvertWorkItemStorageToModel(wiType *WorkItemType, wi *WorkItemStorage) (*WorkItem, error) {
	result, err := wiType.ConvertWorkItemStorageToModel(*wi)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	if _, ok := wiType.Fields[SystemCreatedAt]; ok {
		result.Fields[SystemCreatedAt] = wi.CreatedAt
	}
	if _, ok := wiType.Fields[SystemUpdatedAt]; ok {
		result.Fields[SystemUpdatedAt] = wi.UpdatedAt
	}
	if _, ok := wiType.Fields[SystemOrder]; ok {
		result.Fields[SystemOrder] = wi.ExecutionOrder
	}
	return result, nil

}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormWorkItemRepository) listItemsFromDB(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, limit *int) ([]WorkItemStorage, int, error) {
	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return nil, 0, errors.NewBadParameterError("expression", criteria)
	}
	where = where + " AND space_id = ?"
	parameters = append(parameters, spaceID.String())

	if parentExists != nil && !*parentExists {
		where += ` AND
			id not in (
				SELECT target_id FROM work_item_links
				WHERE link_type_id IN (
					SELECT id FROM work_item_link_types WHERE forward_name = 'parent of'
				)
			)`

	}
	db := r.db.Model(&WorkItemStorage{}).Where(where, parameters...)
	orgDB := db
	if start != nil {
		if *start < 0 {
			return nil, 0, errors.NewBadParameterError("start", *start)
		}
		db = db.Offset(*start)
	}
	if limit != nil {
		if *limit <= 0 {
			return nil, 0, errors.NewBadParameterError("limit", *limit)
		}
		db = db.Limit(*limit)
	}

	db = db.Select("count(*) over () as cnt2 , *").Order("execution_order desc")

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	defer rows.Close()

	result := []WorkItemStorage{}
	columns, err := rows.Columns()
	if err != nil {
		return nil, 0, errors.NewInternalError(ctx, err)
	}

	// need to set up a result for Scan() in order to extract total count.
	var count int
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := WorkItemStorage{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				return nil, 0, errors.NewInternalError(ctx, err)
			}
		}
		result = append(result, value)

	}
	if first {
		// means 0 rows were returned from the first query (maybe becaus of offset outside of total count),
		// need to do a count(*) to find out total
		orgDB := orgDB.Select("count(*)")
		rows2, err := orgDB.Rows()
		defer rows2.Close()
		if err != nil {
			return nil, 0, errs.WithStack(err)
		}
		rows2.Next() // count(*) will always return a row
		rows2.Scan(&count)
	}
	return result, count, nil
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormWorkItemRepository) List(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression, parentExists *bool, start *int, limit *int) ([]WorkItem, int, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "list"}, time.Now())
	result, count, err := r.listItemsFromDB(ctx, spaceID, criteria, parentExists, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	res := make([]WorkItem, len(result))
	for index, value := range result {
		wiType, err := r.witr.LoadTypeFromDB(ctx, value.Type)
		if err != nil {
			return nil, 0, errors.NewInternalError(ctx, err)
		}
		modelWI, err := ConvertWorkItemStorageToModel(wiType, &value)
		if err != nil {
			return nil, 0, errors.NewInternalError(ctx, err)
		}
		res[index] = *modelWI
	}
	return res, count, nil
}

// Count returns the amount of work item that satisfy the given criteria.Expression
func (r *GormWorkItemRepository) Count(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (int, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "count"}, time.Now())

	where, parameters, compileError := Compile(criteria)
	if compileError != nil {
		return 0, errors.NewBadParameterError("expression", criteria)
	}
	where = where + " AND space_id = ?"
	parameters = append(parameters, spaceID)

	var count int
	r.db.Model(&WorkItemStorage{}).Where(where, parameters...).Count(&count)
	return count, nil
}

// Fetch fetches the (first) work item matching by the given criteria.Expression.
func (r *GormWorkItemRepository) Fetch(ctx context.Context, spaceID uuid.UUID, criteria criteria.Expression) (*WorkItem, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "fetch"}, time.Now())

	limit := 1
	results, count, err := r.List(ctx, spaceID, criteria, nil, nil, &limit)
	if err != nil {
		return nil, err
	}
	// if no result
	if count == 0 {
		return nil, nil
	}
	// one result
	result := results[0]
	return &result, nil
}

func (r *GormWorkItemRepository) getAllIterationWithCounts(ctx context.Context, db *gorm.DB, spaceID uuid.UUID) (map[string]WICountsPerIteration, error) {
	var allIterations []uuid.UUID
	db.Pluck("id", &allIterations)
	var res []WICountsPerIteration
	db = r.db.Table(workitemTableName).Select(`iterations.id as IterationId, count(*) as Total,
			count( case fields->>'system.state' when 'closed' then '1' else null end ) as Closed`).Joins(`left join iterations
			on fields@> concat('{"system.iteration": "', iterations.id, '"}')::jsonb`).Where(`iterations.space_id = ?
			and work_items.deleted_at IS NULL`, spaceID).Group(`IterationId`).Scan(&res)
	db.Scan(&res)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID,
			"err":      db.Error,
		}, "unable to count WI for every iteration in a space")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	wiMap := map[string]WICountsPerIteration{}
	for _, r := range res {
		wiMap[r.IterationID] = WICountsPerIteration{
			IterationID: r.IterationID,
			Total:       r.Total,
			Closed:      r.Closed,
		}
	}
	// put 0 count for iterations which are not in wiMap
	// ToDo: Update count query to include non matching rows with 0 values
	// Following operation can be skipped once above is done
	for _, i := range allIterations {
		if _, exists := wiMap[i.String()]; !exists {
			wiMap[i.String()] = WICountsPerIteration{
				IterationID: i.String(),
				Total:       0,
				Closed:      0,
			}
		}
	}
	return wiMap, nil
}

func (r *GormWorkItemRepository) getFinalCountAddingChild(ctx context.Context, db *gorm.DB, spaceID uuid.UUID, wiMap map[string]WICountsPerIteration) (map[string]WICountsPerIteration, error) {
	iterationTable := iteration.Iteration{}
	iterationTableName := iterationTable.TableName()
	type IterationHavingChildrenID struct {
		Children    string `gorm:"column:children"`
		IterationID string `gorm:"column:iterationid"`
	}
	var itrChildren []IterationHavingChildrenID
	queryIterationWithChildren := fmt.Sprintf(`
	WITH PathResolver AS
	(SELECT CASE
				WHEN path = '' THEN replace(id::text, '-', '_')::ltree
				ELSE concat(path::text, '.', REPLACE(id::text, '-', '_'))::ltree
			END AS pathself,
			id
	FROM %s
	WHERE space_id = ?)
	SELECT array_agg(iterations.id)::text AS children,
		PathResolver.id::text AS iterationid
	FROM %s,
		PathResolver
	WHERE path <@ PathResolver.pathself
	AND space_id = ?
	GROUP BY (PathResolver.pathself,
		PathResolver.id)`,
		iterationTableName,
		iterationTableName)
	db = r.db.Raw(queryIterationWithChildren, spaceID.String(), spaceID.String())
	db.Scan(&itrChildren)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": spaceID.String(),
			"err":      db.Error,
		}, "unable to fetch children for every iteration in a space")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	childMap := map[string][]string{}
	for _, r := range itrChildren {
		// Following can be done by implementing Valuer interface for type IterationHavingChildrenID
		r.Children = strings.TrimPrefix(r.Children, "{")
		r.Children = strings.TrimRight(r.Children, "}")
		children := strings.Split(r.Children, ",")
		childMap[r.IterationID] = children
	}
	countsMap := map[string]WICountsPerIteration{}
	for _, i := range wiMap {
		t := i.Total
		c := i.Closed
		if children, exists := childMap[i.IterationID]; exists {
			for _, child := range children {
				if _, exists := wiMap[child]; exists {
					t += wiMap[child].Total
					c += wiMap[child].Closed
				}
			}
		}
		countsMap[i.IterationID] = WICountsPerIteration{
			IterationID: i.IterationID,
			Total:       t,
			Closed:      c,
		}
	}
	return countsMap, nil
}

// GetCountsPerIteration counts WIs including iteration-children and returns a map of iterationID->WICountsPerIteration
func (r *GormWorkItemRepository) GetCountsPerIteration(ctx context.Context, spaceID uuid.UUID) (map[string]WICountsPerIteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "getCountsPerIteration"}, time.Now())
	db := r.db.Model(&iteration.Iteration{}).Where("space_id = ?", spaceID)
	if db.Error != nil {
		return nil, errors.NewInternalError(ctx, db.Error)
	}

	wiMap, err := r.getAllIterationWithCounts(ctx, db, spaceID)
	if err != nil {
		return nil, err
	}

	countsMap, err := r.getFinalCountAddingChild(ctx, db, spaceID, wiMap)
	if err != nil {
		return nil, err
	}
	return countsMap, nil
}

// GetCountsForIteration returns Closed and Total counts of WIs for given iteration
// It fetches all child iterations of input iteration and then uses list to counts work items
// SELECT count(*) AS Total,
//        count(CASE fields->>'system.state'
//                  WHEN 'closed' THEN '1'
//                  ELSE NULL
//              END) AS Closed
// FROM work_items wi
// WHERE fields->>'system.iteration' IN ('input iteration ID + children IDs')
//   AND wi.deleted_at IS NULL
func (r *GormWorkItemRepository) GetCountsForIteration(ctx context.Context, itr *iteration.Iteration) (map[string]WICountsPerIteration, error) {
	defer goa.MeasureSince([]string{"goa", "db", "workitem", "getCountsForIteration"}, time.Now())
	var res WICountsPerIteration
	pathOfIteration := append(itr.Path, itr.ID)
	// get child IDs of the iteration
	var childIDs []uuid.UUID
	iterationTable := iteration.Iteration{}
	iterationTableName := iterationTable.TableName()
	getIterationsOfSpace := fmt.Sprintf(`SELECT id FROM %s WHERE path <@ ? and space_id = ?`, iterationTableName)
	db := r.db.Raw(getIterationsOfSpace, pathOfIteration.Convert(), itr.SpaceID.String())
	db.Pluck("id", &childIDs)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"path": pathOfIteration.Convert(),
			"err":  db.Error,
		}, "unable to fetch children for path")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	childIDs = append(childIDs, itr.ID)

	// build where clause usig above ID list
	idsToLookFor := []string{}
	for _, x := range childIDs {
		partialClause := fmt.Sprintf(`fields @> '{"system.iteration":"%s"}'`, x.String())
		idsToLookFor = append(idsToLookFor, partialClause)
	}
	whereClause := strings.Join(idsToLookFor, " OR ")
	query := fmt.Sprintf(`SELECT count(*) AS Total,
						count(CASE fields->>'system.state'
									WHEN 'closed' THEN '1'
									ELSE NULL
								END) AS Closed
					FROM %s wi
					WHERE %s
					AND wi.deleted_at IS NULL`,
		workitemTableName, whereClause)
	db = r.db.Raw(query)
	db.Scan(&res)
	if db.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"iteration_id`": whereClause,
			"err":           db.Error,
		}, "unable to count WI for an iteration")
		return nil, errors.NewInternalError(ctx, db.Error)
	}
	countsMap := map[string]WICountsPerIteration{}
	countsMap[itr.ID.String()] = WICountsPerIteration{
		IterationID: itr.ID.String(),
		Closed:      res.Closed,
		Total:       res.Total,
	}
	return countsMap, nil
}
