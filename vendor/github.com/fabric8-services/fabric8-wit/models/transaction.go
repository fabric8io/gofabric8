package models

import (
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
)

// Transactional executes the given function in a transaction. If todo returns an error, the transaction is rolled back
func Transactional(db *gorm.DB, todo func(tx *gorm.DB) error) error {
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := todo(tx); err != nil {
		tx.Rollback()
		return errs.WithStack(err)
	}
	tx.Commit()
	return tx.Error
}
