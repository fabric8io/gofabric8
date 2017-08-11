package space

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	SystemSpace = uuid.FromStringOrNil("2e0698d8-753e-4cef-bb7c-f027634824a2")
	SpaceType   = "spaces"
)

// Space represents a Space on the domain and db layer
type Space struct {
	gormsupport.Lifecycle
	ID          uuid.UUID
	Version     int
	Name        string
	Description string
	OwnerId     uuid.UUID `sql:"type:uuid"` // Belongs To Identity
}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = Space{}
var _ convert.Equaler = (*Space)(nil)

// Equal returns true if two Space objects are equal; otherwise false is returned.
func (p Space) Equal(u convert.Equaler) bool {
	other, ok := u.(Space)
	if !ok {
		return false
	}
	lfEqual := p.Lifecycle.Equal(other.Lifecycle)
	if !lfEqual {
		return false
	}
	if p.Version != other.Version {
		return false
	}
	if p.Name != other.Name {
		return false
	}
	if p.Description != other.Description {
		return false
	}
	if !uuid.Equal(p.OwnerId, other.OwnerId) {
		return false
	}
	return true
}

// GetETagData returns the field values to use to generate the ETag
func (p Space) GetETagData() []interface{} {
	return []interface{}{p.ID, p.Version}
}

// GetLastModified returns the last modification time
func (p Space) GetLastModified() time.Time {
	return p.UpdatedAt
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (p Space) TableName() string {
	return "spaces"
}

// Repository encapsulate storage & retrieval of spaces
type Repository interface {
	repository.Exister
	Create(ctx context.Context, space *Space) (*Space, error)
	Save(ctx context.Context, space *Space) (*Space, error)
	Load(ctx context.Context, ID uuid.UUID) (*Space, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	LoadByOwner(ctx context.Context, userID *uuid.UUID, start *int, length *int) ([]Space, uint64, error)
	LoadByOwnerAndName(ctx context.Context, userID *uuid.UUID, spaceName *string) (*Space, error)
	List(ctx context.Context, start *int, length *int) ([]Space, uint64, error)
	Search(ctx context.Context, q *string, start *int, length *int) ([]Space, uint64, error)
}

// NewRepository creates a new space repo
func NewRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db}
}

// GormRepository implements SpaceRepository using gorm
type GormRepository struct {
	db *gorm.DB
}

// Load returns the space for the given id
// returns NotFoundError or InternalError
func (r *GormRepository) Load(ctx context.Context, ID uuid.UUID) (*Space, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "load"}, time.Now())
	res := Space{}
	tx := r.db.Where("id=?", ID).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"space_id": ID.String(),
		}, "state or known referer was empty")
		return nil, errors.NewNotFoundError("space", ID.String())
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      tx.Error,
			"space_id": ID.String(),
		}, "unable to load the space by ID")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res, nil
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (r *GormRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "space", "exists"}, time.Now())
	return repository.CheckExists(ctx, r.db, Space{}.TableName(), id)
}

// Delete deletes the space with the given id
// returns NotFoundError or InternalError
func (r *GormRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "space", "delete"}, time.Now())
	if ID == uuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": ID.String(),
		}, "unable to find the space by ID")
		return errors.NewNotFoundError("space", ID.String())
	}
	space := Space{ID: ID}
	tx := r.db.Delete(space)

	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"space_id": ID.String(),
		}, "unable to delete the space")
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"space_id": ID.String(),
		}, "none row was affected by the deletion operation")
		return errors.NewNotFoundError("space", ID.String())
	}

	return nil
}

// Save updates the given space in the db. Version must be the same as the one in the stored version
// returns NotFoundError, BadParameterError, VersionConflictError or InternalError
func (r *GormRepository) Save(ctx context.Context, p *Space) (*Space, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "save"}, time.Now())
	pr := Space{}
	tx := r.db.Where("id=?", p.ID).First(&pr)
	oldVersion := p.Version
	p.Version++
	if tx.RecordNotFound() {
		// treating this as a not found error: the fact that we're using number internal is implementation detail
		return nil, errors.NewNotFoundError("space", p.ID.String())
	}
	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":      tx.Error,
			"space_id": p.ID,
		}, "unable to find the space by ID")
		return nil, errors.NewInternalError(ctx, err)
	}
	tx = tx.Where("Version = ?", oldVersion).Save(p)
	if err := tx.Error; err != nil {
		if gormsupport.IsCheckViolation(tx.Error, "spaces_name_check") {
			return nil, errors.NewBadParameterError("Name", p.Name).Expected("not empty")
		}
		if gormsupport.IsUniqueViolation(tx.Error, "spaces_name_idx") {
			return nil, errors.NewBadParameterError("Name", p.Name).Expected("unique")
		}
		log.Error(ctx, map[string]interface{}{
			"err":      err,
			"version":  oldVersion,
			"space_id": p.ID,
		}, "unable to find the space by version")
		return nil, errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		return nil, errors.NewVersionConflictError("version conflict")
	}

	log.Info(ctx, map[string]interface{}{
		"space_id": p.ID,
	}, "space updated successfully")
	return p, nil
}

// Create creates a new Space in the db
// returns BadParameterError or InternalError
func (r *GormRepository) Create(ctx context.Context, space *Space) (*Space, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "create"}, time.Now())
	// We might want to create a space with a specific ID, e.g. space.SystemSpace
	if space.ID == uuid.Nil {
		space.ID = uuid.NewV4()
	}

	tx := r.db.Create(space)
	if err := tx.Error; err != nil {
		if gormsupport.IsCheckViolation(tx.Error, "spaces_name_check") {
			return nil, errors.NewBadParameterError("Name", space.Name).Expected("not empty")
		}
		if gormsupport.IsUniqueViolation(tx.Error, "spaces_name_idx") {
			log.Error(ctx, map[string]interface{}{
				"err":        err,
				"space_name": space.Name,
			}, "unable to create space because a space with the same name already exists for this user")
			return nil, errors.NewDataConflictError(fmt.Sprintf("space already exists ( for this user ) : %s ", space.Name))
		}
		return nil, errors.NewInternalError(ctx, err)
	}

	log.Info(ctx, map[string]interface{}{
		"space_id": space.ID,
	}, "Space created successfully")
	return space, nil
}

// extracted this function from List() in order to close the rows object with "defer" for more readability
// workaround for https://github.com/lib/pq/issues/81
func (r *GormRepository) listSpaceFromDB(ctx context.Context, q *string, userID *uuid.UUID, start *int, limit *int) ([]Space, uint64, error) {
	db := r.db.Model(&Space{})
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
	db = db.Select("count(*) over () as cnt2 , *")
	if q != nil {
		db = db.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(*q)+"%")
		db = db.Or("LOWER(description) LIKE ?", "%"+strings.ToLower(*q)+"%")
	}
	if userID != nil {
		db = db.Where("spaces.owner_id=?", userID)
	}

	rows, err := db.Rows()
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	defer rows.Close()

	result := []Space{}
	columns, err := rows.Columns()
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err":   err,
			"query": q,
		}, "unable to load spaces")
		return nil, 0, errors.NewInternalError(ctx, err)
	}

	// need to set up a result for Scan() in order to extract total count.
	var count uint64
	var ignore interface{}
	columnValues := make([]interface{}, len(columns))

	for index := range columnValues {
		columnValues[index] = &ignore
	}
	columnValues[0] = &count
	first := true

	for rows.Next() {
		value := Space{}
		db.ScanRows(rows, &value)
		if first {
			first = false
			if err = rows.Scan(columnValues...); err != nil {
				log.Error(ctx, map[string]interface{}{
					"err": err,
				}, "unable to load spaces")
				return nil, 0, errors.NewInternalError(ctx, err)
			}
		}
		result = append(result, value)
	}
	if first {
		if q != nil {
			// If 0 rows were returned from first query during search, then total is 0
			count = 0
		} else {
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
	}
	return result, count, nil
}

// List returns work item selected by the given criteria.Expression, starting with start (zero-based) and returning at most limit items
func (r *GormRepository) List(ctx context.Context, start *int, limit *int) ([]Space, uint64, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "list"}, time.Now())
	result, count, err := r.listSpaceFromDB(ctx, nil, nil, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	return result, count, nil
}

func (r *GormRepository) Search(ctx context.Context, q *string, start *int, limit *int) ([]Space, uint64, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "search"}, time.Now())
	result, count, err := r.listSpaceFromDB(ctx, q, nil, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	return result, count, nil
}

func (r *GormRepository) LoadByOwner(ctx context.Context, userID *uuid.UUID, start *int, limit *int) ([]Space, uint64, error) {
	result, count, err := r.listSpaceFromDB(ctx, nil, userID, start, limit)
	if err != nil {
		return nil, 0, errs.WithStack(err)
	}
	return result, count, nil
}

func (r *GormRepository) LoadByOwnerAndName(ctx context.Context, userID *uuid.UUID, spaceName *string) (*Space, error) {
	defer goa.MeasureSince([]string{"goa", "db", "space", "loadByOwnerAndName"}, time.Now())
	res := Space{}
	tx := r.db.Where("spaces.owner_id=? AND LOWER(spaces.name)=?", *userID, strings.ToLower(*spaceName)).First(&res)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"space_name": *spaceName,
			"user_id":    *userID,
		}, "Could not find space under owner")
		return nil, errors.NewNotFoundError("space", *spaceName)
	}
	if tx.Error != nil {
		log.Error(ctx, map[string]interface{}{
			"err":        tx.Error,
			"space_name": *spaceName,
			"user_id":    *userID,
		}, "unable to load space by owner and name")
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &res, nil
}
