package account

import (
	"context"
	"strconv"
	"time"

	"github.com/fabric8-services/fabric8-wit/application/repository"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// In future, we could add support for FieldDefinitions the way we have for workitems.
// Hence. keeping the map as a string->interface and not string->string.
// At the moment, FieldDefinitions could be an overkill, so keeping it out.

// User describes a User account. A few identities can be assosiated with one user account
type User struct {
	gormsupport.Lifecycle
	ID                 uuid.UUID          `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"` // This is the ID PK field
	Email              string             `sql:"unique_index"`                                            // This is the unique email field
	FullName           string             // The fullname of the User
	ImageURL           string             // The image URL for the User
	Bio                string             // The bio of the User
	URL                string             // The URL of the User
	Company            string             // The (optional) Company of the User
	Identities         []Identity         // has many Identities from different IDPs
	ContextInformation ContextInformation `sql:"type:jsonb"` // context information of the user activity
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m User) TableName() string {
	return "users"
}

// GetETagData returns the field values to use to generate the ETag
func (m User) GetETagData() []interface{} {
	// using the 'ID' and 'UpdatedAt' (converted to number of seconds since epoch) fields
	return []interface{}{m.ID, strconv.FormatInt(m.UpdatedAt.Unix(), 10)}
}

// GetLastModified returns the last modification time
func (m User) GetLastModified() time.Time {
	return m.UpdatedAt
}

// GormUserRepository is the implementation of the storage interface for User.
type GormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new storage type.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &GormUserRepository{db: db}
}

// UserRepository represents the storage interface.
type UserRepository interface {
	repository.Exister
	Load(ctx context.Context, ID uuid.UUID) (*User, error)
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	List(ctx context.Context) ([]User, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Query(funcs ...func(*gorm.DB) *gorm.DB) ([]User, error)
}

// TableName overrides the table name settings in Gorm to force a specific table name
// in the database.
func (m *GormUserRepository) TableName() string {
	return "users"
}

// CRUD Functions

// Load returns a single User as a Database Model
// This is more for use internally, and probably not what you want in  your controllers
func (m *GormUserRepository) Load(ctx context.Context, id uuid.UUID) (*User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "load"}, time.Now())
	var native User
	err := m.db.Table(m.TableName()).Where("id = ?", id).Find(&native).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.NewNotFoundError("user", id.String())
	}
	return &native, errs.WithStack(err)
}

// CheckExists returns nil if the given ID exists otherwise returns an error
func (m *GormUserRepository) CheckExists(ctx context.Context, id string) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "exists"}, time.Now())
	return repository.CheckExists(ctx, m.db, m.TableName(), id)
}

// Create creates a new record.
func (m *GormUserRepository) Create(ctx context.Context, u *User) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "create"}, time.Now())
	if u.ID == uuid.Nil {
		u.ID = uuid.NewV4()
	}
	err := m.db.Create(u).Error
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"user_id": u.ID,
			"err":     err,
		}, "unable to create the user")
		return errs.WithStack(err)
	}
	log.Debug(ctx, map[string]interface{}{
		"user_id": u.ID,
	}, "User created!")
	return nil
}

// Save modifies a single record
func (m *GormUserRepository) Save(ctx context.Context, model *User) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "save"}, time.Now())

	obj, err := m.Load(ctx, model.ID)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"user_id": model.ID,
			"err":     err,
		}, "unable to update user")
		return errs.WithStack(err)
	}
	err = m.db.Model(obj).Updates(model).Error
	if err != nil {
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"user_id": model.ID,
	}, "User saved!")
	return nil
}

// Delete removes a single record.
func (m *GormUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	defer goa.MeasureSince([]string{"goa", "db", "user", "delete"}, time.Now())

	obj := User{ID: id}

	err := m.db.Delete(&obj).Error

	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"user_id": id,
			"err":     err,
		}, "unable to delete the user")
		return errs.WithStack(err)
	}

	log.Debug(ctx, map[string]interface{}{
		"user_id": id,
	}, "User deleted!")

	return nil
}

// List return all users
func (m *GormUserRepository) List(ctx context.Context) ([]User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "list"}, time.Now())
	var rows []User

	err := m.db.Model(&User{}).Order("email").Find(&rows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}
	return rows, nil
}

// Query expose an open ended Query model
func (m *GormUserRepository) Query(funcs ...func(*gorm.DB) *gorm.DB) ([]User, error) {
	defer goa.MeasureSince([]string{"goa", "db", "user", "query"}, time.Now())
	var objs []User

	err := m.db.Scopes(funcs...).Table(m.TableName()).Find(&objs).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errs.WithStack(err)
	}

	log.Debug(nil, map[string]interface{}{
		"user_list": objs,
	}, "User query done successfully!")

	return objs, nil
}

// UserFilterByID is a gorm filter for User ID.
func UserFilterByID(userID uuid.UUID) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", userID)
	}
}

// UserFilterByEmail is a gorm filter for User ID.
func UserFilterByEmail(email string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ?", email)
	}
}
