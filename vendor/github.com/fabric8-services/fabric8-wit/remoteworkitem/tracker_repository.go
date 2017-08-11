package remoteworkitem

import (
	"strconv"

	"fmt"

	"context"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	govalidator "gopkg.in/asaskevich/govalidator.v4"
)

const trackersTableName = "trackers"

// GormTrackerRepository implements TrackerRepository using gorm
type GormTrackerRepository struct {
	db *gorm.DB
}

// NewTrackerRepository constructs a TrackerRepository
func NewTrackerRepository(db *gorm.DB) *GormTrackerRepository {
	return &GormTrackerRepository{db}
}

// Create creates a new tracker configuration in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerRepository) Create(ctx context.Context, url string, typeID string) (*app.Tracker, error) {
	//URL Validation
	isValid := govalidator.IsURL(url)
	if isValid != true {
		return nil, BadParameterError{parameter: "url", value: url}
	}

	_, present := RemoteWorkItemImplRegistry[typeID]
	// Ensure we support this remote tracker.
	if present != true {
		return nil, BadParameterError{parameter: "type", value: typeID}
	}
	t := Tracker{
		URL:  url,
		Type: typeID}
	tx := r.db
	if err := tx.Create(&t).Error; err != nil {
		return nil, InternalError{simpleError{err.Error()}}
	}
	log.Info(ctx, map[string]interface{}{
		"tracker": t,
	}, "Tracker reposity created")

	t2 := app.Tracker{
		ID:   strconv.FormatUint(t.ID, 10),
		URL:  url,
		Type: typeID}

	return &t2, nil
}

// Load returns the tracker configuration for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Load(ctx context.Context, ID string) (*app.Tracker, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", ID}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_id": id,
	}, "Loading tracker repository...")

	res := Tracker{}
	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": ID,
		}, "tracker repository not found")

		return nil, NotFoundError{"tracker", ID}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{fmt.Sprintf("error while loading: %s", tx.Error.Error())}}
	}
	t := app.Tracker{
		ID:   strconv.FormatUint(res.ID, 10),
		URL:  res.URL,
		Type: res.Type}

	return &t, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormTrackerRepository) CheckExists(ctx context.Context, id string) error {
	return repository.CheckExists(ctx, r.db, trackersTableName, id)
}

// List returns tracker selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerRepository) List(ctx context.Context, criteria criteria.Expression, start *int, limit *int) ([]*app.Tracker, error) {
	where, parameters, err := workitem.Compile(criteria)
	if err != nil {
		return nil, BadParameterError{"expression", criteria}
	}

	log.Info(ctx, map[string]interface{}{
		"query": where,
	}, "Executing tracker repository query...")

	var rows []Tracker
	db := r.db.Where(where, parameters...)
	if start != nil {
		db = db.Offset(*start)
	}
	if limit != nil {
		db = db.Limit(*limit)
	}
	if err := db.Find(&rows).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]*app.Tracker, len(rows))

	for i, tracker := range rows {
		t := app.Tracker{
			ID:   strconv.FormatUint(tracker.ID, 10),
			URL:  tracker.URL,
			Type: tracker.Type}
		result[i] = &t
	}
	return result, nil
}

// Save updates the given tracker in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerRepository) Save(ctx context.Context, t app.Tracker) (*app.Tracker, error) {
	res := Tracker{}
	id, err := strconv.ParseUint(t.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, NotFoundError{entity: "tracker", ID: t.ID}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_id": id,
	}, "Looking for a tracker repository with id ", id)

	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": id,
		}, "tracker repository not found")

		return nil, NotFoundError{entity: "tracker", ID: t.ID}
	}
	_, present := RemoteWorkItemImplRegistry[t.Type]
	// Ensure we support this remote tracker.
	if present != true {
		return nil, BadParameterError{parameter: "type", value: t.Type}
	}

	newT := Tracker{
		ID:   id,
		URL:  t.URL,
		Type: t.Type}

	if err := tx.Save(&newT).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": newT.ID,
			"err":        err,
		}, "unable to save tracker repository")
		return nil, InternalError{simpleError{err.Error()}}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker": newT.ID,
	}, "Tracker repository successfully updated")

	t2 := app.Tracker{
		ID:   strconv.FormatUint(id, 10),
		URL:  t.URL,
		Type: t.Type}

	return &t2, nil
}

// Delete deletes the tracker with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerRepository) Delete(ctx context.Context, ID string) error {
	var t = Tracker{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treat as not found: clients don't know it must be a number
		return NotFoundError{entity: "tracker", ID: ID}
	}
	t.ID = id
	tx := r.db.Delete(t)
	if err = tx.Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return NotFoundError{entity: "tracker", ID: ID}
	}
	return nil
}
