package auth

import (
	"github.com/fabric8-services/fabric8-wit/convert"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"

	"context"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

const (
	oauthStateTableName = "oauth_state_references"
)

// OauthStateReference represents a oauth state reference
type OauthStateReference struct {
	gormsupport.Lifecycle
	ID       uuid.UUID `sql:"type:uuid default uuid_generate_v4()" gorm:"primary_key"`
	Referrer string
}

// TableName implements gorm.tabler
func (r OauthStateReference) TableName() string {
	return oauthStateTableName
}

// Equal returns true if two States objects are equal; otherwise false is returned.
func (r OauthStateReference) Equal(u convert.Equaler) bool {
	other, ok := u.(OauthStateReference)
	if !ok {
		return false
	}
	if r.ID != other.ID {
		return false
	}
	if r.Referrer != other.Referrer {
		return false
	}
	return true
}

// OauthStateReferenceRepository encapsulate storage & retrieval of state references
type OauthStateReferenceRepository interface {
	Create(ctx context.Context, state *OauthStateReference) (*OauthStateReference, error)
	Delete(ctx context.Context, ID uuid.UUID) error
	Load(ctx context.Context, ID uuid.UUID) (*OauthStateReference, error)
}

// NewOauthStateReferenceRepository creates a new oauth state reference repo
func NewOauthStateReferenceRepository(db *gorm.DB) *GormOauthStateReferenceRepository {
	return &GormOauthStateReferenceRepository{db}
}

// GormOauthStateReferenceRepository implements OauthStateReferenceRepository using gorm
type GormOauthStateReferenceRepository struct {
	db *gorm.DB
}

// Delete deletes the reference with the given id
// returns NotFoundError or InternalError
func (r *GormOauthStateReferenceRepository) Delete(ctx context.Context, ID uuid.UUID) error {
	if ID == uuid.Nil {
		log.Error(ctx, map[string]interface{}{
			"oauth_state_reference_id": ID.String(),
		}, "unable to find the oauth state reference by ID")
		return errors.NewNotFoundError("oauth state reference", ID.String())
	}
	reference := OauthStateReference{ID: ID}
	tx := r.db.Delete(reference)

	if err := tx.Error; err != nil {
		log.Error(ctx, map[string]interface{}{
			"oauth_state_reference_id": ID.String(),
		}, "unable to delete the oauth state reference")
		return errors.NewInternalError(ctx, err)
	}
	if tx.RowsAffected == 0 {
		log.Error(ctx, map[string]interface{}{
			"oauth state reference": ID.String(),
		}, "none row was affected by the deletion operation")
		return errors.NewNotFoundError("oauth state reference", ID.String())
	}

	return nil
}

// Create creates a new oauth state reference in the DB
// returns InternalError
func (r *GormOauthStateReferenceRepository) Create(ctx context.Context, reference *OauthStateReference) (*OauthStateReference, error) {
	if reference.ID == uuid.Nil {
		reference.ID = uuid.NewV4()
	}

	tx := r.db.Create(reference)
	if err := tx.Error; err != nil {
		return nil, errors.NewInternalError(ctx, err)
	}

	log.Info(ctx, map[string]interface{}{
		"oauth_state_reference_id": reference.ID,
	}, "Oauth state reference created successfully")
	return reference, nil
}

// Load loads state reference by ID
func (r *GormOauthStateReferenceRepository) Load(ctx context.Context, id uuid.UUID) (*OauthStateReference, error) {
	ref := OauthStateReference{}
	tx := r.db.Where("id=?", id).First(&ref)
	if tx.RecordNotFound() {
		log.Error(ctx, map[string]interface{}{
			"id": id.String(),
		}, "Could not find oauth state reference by state")
		return nil, errors.NewNotFoundError("oauth state reference", id.String())
	}
	if tx.Error != nil {
		return nil, errors.NewInternalError(ctx, tx.Error)
	}
	return &ref, nil
}
