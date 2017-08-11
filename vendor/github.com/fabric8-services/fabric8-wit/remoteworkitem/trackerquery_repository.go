package remoteworkitem

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const trackerQueriesTableName = "tracker_queries"

// GormTrackerQueryRepository implements TrackerRepository using gorm
type GormTrackerQueryRepository struct {
	db *gorm.DB
}

// NewTrackerQueryRepository constructs a TrackerQueryRepository
func NewTrackerQueryRepository(db *gorm.DB) *GormTrackerQueryRepository {
	return &GormTrackerQueryRepository{db}
}

// Create creates a new tracker query in the repository
// returns BadParameterError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Create(ctx context.Context, query string, schedule string, tracker string, spaceID uuid.UUID) (*app.TrackerQuery, error) {
	tid, err := strconv.ParseUint(tracker, 10, 64)
	if err != nil || tid == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", tracker}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_id": tid,
	}, "Tracker ID to be created")

	tq := TrackerQuery{
		Query:     query,
		Schedule:  schedule,
		TrackerID: tid,
		SpaceID:   spaceID,
	}
	tx := r.db
	if err := tx.Create(&tq).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"tracker_id":    tid,
			"tracker_query": query,
		}, "unable to create the tracker query")
		return nil, InternalError{simpleError{err.Error()}}
	}

	spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(spaceID.String()))
	tq2 := app.TrackerQuery{
		ID:        strconv.FormatUint(tq.ID, 10),
		Query:     query,
		Schedule:  schedule,
		TrackerID: tracker,
		Relationships: &app.TrackerQueryRelationships{
			Space: app.NewSpaceRelation(spaceID, spaceSelfURL),
		},
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_id":    tid,
		"tracker_query": tq,
	}, "Created tracker query")

	return &tq2, nil
}

// Load returns the tracker query for the given id
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Load(ctx context.Context, ID string) (*app.TrackerQuery, error) {
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker query", ID}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_query_id": id,
	}, "Loading the tracker query")

	res := TrackerQuery{}
	if r.db.First(&res, id).RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": id,
		}, "tracker resource not found")
		return nil, NotFoundError{"tracker query", ID}
	}

	spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(res.SpaceID.String()))
	tq := app.TrackerQuery{
		ID:        strconv.FormatUint(res.ID, 10),
		Query:     res.Query,
		Schedule:  res.Schedule,
		TrackerID: strconv.FormatUint(res.TrackerID, 10),
		Relationships: &app.TrackerQueryRelationships{
			Space: app.NewSpaceRelation(res.SpaceID, spaceSelfURL),
		},
	}

	return &tq, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormTrackerQueryRepository) CheckExists(ctx context.Context, id string) error {
	return repository.CheckExists(ctx, r.db, trackerQueriesTableName, id)
}

// Save updates the given tracker query in storage.
// returns NotFoundError, ConversionError or InternalError
func (r *GormTrackerQueryRepository) Save(ctx context.Context, tq app.TrackerQuery) (*app.TrackerQuery, error) {
	res := TrackerQuery{}
	id, err := strconv.ParseUint(tq.ID, 10, 64)
	if err != nil || id == 0 {
		return nil, NotFoundError{entity: "trackerquery", ID: tq.ID}
	}

	tid, err := strconv.ParseUint(tq.TrackerID, 10, 64)
	if err != nil || tid == 0 {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, NotFoundError{"tracker", tq.TrackerID}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_id": id,
	}, "looking tracker query")

	tx := r.db.First(&res, id)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": id,
		}, "tracker query not found")

		return nil, NotFoundError{entity: "TrackerQuery", ID: tq.ID}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{fmt.Sprintf("could not load tracker query: %s", tx.Error.Error())}}
	}

	tx = r.db.First(&Tracker{}, tid)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"tracker_id": id,
		}, "tracker ID not found")
		return nil, NotFoundError{entity: "tracker", ID: tq.TrackerID}
	}
	if tx.Error != nil {
		return nil, InternalError{simpleError{fmt.Sprintf("could not load tracker: %s", tx.Error.Error())}}
	}

	newTq := TrackerQuery{
		ID:        id,
		Schedule:  tq.Schedule,
		Query:     tq.Query,
		TrackerID: tid,
		SpaceID:   *tq.Relationships.Space.Data.ID,
	}

	if err := tx.Save(&newTq).Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"tracker_query": tq.Query,
			"tracker_id":    tid,
			"err":           err,
		}, "unable to save the tracker query")
		return nil, InternalError{simpleError{err.Error()}}
	}

	log.Info(ctx, map[string]interface{}{
		"tracker_query": newTq,
	}, "Updated tracker query")

	spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(tq.Relationships.Space.Data.ID.String()))
	t2 := app.TrackerQuery{
		ID:        tq.ID,
		Schedule:  tq.Schedule,
		Query:     tq.Query,
		TrackerID: tq.TrackerID,
		Relationships: &app.TrackerQueryRelationships{
			Space: app.NewSpaceRelation(*tq.Relationships.Space.Data.ID, spaceSelfURL),
		},
	}

	return &t2, nil
}

// Delete deletes the tracker query with the given id
// returns NotFoundError or InternalError
func (r *GormTrackerQueryRepository) Delete(ctx context.Context, ID string) error {
	var tq = TrackerQuery{}
	id, err := strconv.ParseUint(ID, 10, 64)
	if err != nil || id == 0 {
		// treat as not found: clients don't know it must be a number
		return NotFoundError{entity: "trackerquery", ID: ID}
	}
	tq.ID = id
	tx := r.db
	tx = tx.Delete(tq)
	if err = tx.Error; err != nil {
		return InternalError{simpleError{err.Error()}}
	}
	if tx.RowsAffected == 0 {
		return NotFoundError{entity: "trackerquery", ID: ID}
	}
	return nil
}

// List returns tracker query selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormTrackerQueryRepository) List(ctx context.Context) ([]*app.TrackerQuery, error) {
	var rows []TrackerQuery
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]*app.TrackerQuery, len(rows))

	for i, tq := range rows {
		spaceSelfURL := rest.AbsoluteURL(goa.ContextRequest(ctx), app.SpaceHref(tq.SpaceID.String()))
		t := app.TrackerQuery{
			ID:        strconv.FormatUint(tq.ID, 10),
			Schedule:  tq.Schedule,
			Query:     tq.Query,
			TrackerID: strconv.FormatUint(tq.TrackerID, 10),
			Relationships: &app.TrackerQueryRelationships{
				Space: app.NewSpaceRelation(tq.SpaceID, spaceSelfURL),
			},
		}
		result[i] = &t
	}
	return result, nil
}
