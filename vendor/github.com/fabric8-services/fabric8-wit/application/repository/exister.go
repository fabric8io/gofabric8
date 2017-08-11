package repository

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/errors"

	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

type Exister interface {
	// Exists returns nil if the object with the given ID exists;
	// otherwise an error is returned in case the given ID doesn't exists or any
	// other unknown issue occured
	CheckExists(ctx context.Context, id string) error
}

// Exists returns true if an item exists in the database table with a given ID
func Exists(ctx context.Context, db *gorm.DB, tableName string, id string) (bool, error) {
	var exists bool
	query := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %[1]s
			WHERE
				id=$1
				AND deleted_at IS NULL
		)`, tableName)

	err := db.CommonDB().QueryRow(query, id).Scan(&exists)
	if err == nil && !exists {
		return exists, errors.NewNotFoundError(tableName, id)
	}
	if err != nil {
		return false, errors.NewInternalError(ctx, errs.Wrapf(err, "unable to verify if %s exists", tableName))
	}
	return exists, nil
}

// CheckExists does the same as Exists but only returns the error value; thereby
// being a handy convenience function.
func CheckExists(ctx context.Context, db *gorm.DB, tableName string, id string) error {
	_, err := Exists(ctx, db, tableName, id)
	return err
}
