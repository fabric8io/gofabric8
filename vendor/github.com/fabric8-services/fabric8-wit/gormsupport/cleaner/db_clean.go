package cleaner

import (
	"database/sql"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/workitem"
	uuid "github.com/satori/go.uuid"

	"fmt"

	"github.com/jinzhu/gorm"
)

// DeleteCreatedEntities records all created entities on the gorm.DB connection
// and returns a function which can be called on defer to delete created
// entities in reverse order on function exit.
//
// In addition to that, the WIT cache is cleared as well in order to respect any
// deletions made to the db.
//
// Usage:
//
// func TestDatabaseActions(t *testing.T) {
//
// 	// setup database connection
// 	db := ....
// 	// setup auto clean up of created entities
// 	defer DeleteCreatedEntities(db)()
//
// 	repo := NewRepo(db)
// 	repo.Create(X)
// 	repo.Create(X)
// 	repo.Create(X)
// }
//
// Output:
//
// 2017/01/31 12:08:08 Deleting from x 6d143405-1232-40de-bc73-835b543cd972
// 2017/01/31 12:08:08 Deleting from x 0685068d-4934-4d9a-bac2-91eebbca9575
// 2017/01/31 12:08:08 Deleting from x 2d20944e-7952-40c1-bd15-f3fa1a70026d
func DeleteCreatedEntities(db *gorm.DB) func() {
	hookName := "mighti:record"
	type entity struct {
		table   string
		keyname string
		key     interface{}
	}
	var entires []entity
	hookRegistered := db.Callback().Create().Get(hookName) != nil
	if hookRegistered {
		hookName += "-" + uuid.NewV4().String()
	}
	db.Callback().Create().After("gorm:create").Register(hookName, func(scope *gorm.Scope) {
		log.Logger().Debugln(fmt.Sprintf("Inserted entities from %s with %s=%v", scope.TableName(), scope.PrimaryKey(), scope.PrimaryKeyValue()))
		entires = append(entires, entity{table: scope.TableName(), keyname: scope.PrimaryKey(), key: scope.PrimaryKeyValue()})
	})
	return func() {
		defer db.Callback().Create().Remove(hookName)
		// Find out if the current db object is already a transaction
		_, inTransaction := db.CommonDB().(*sql.Tx)
		tx := db
		if !inTransaction {
			tx = db.Begin()
		}
		for i := len(entires) - 1; i >= 0; i-- {
			entry := entires[i]
			log.Info(nil, map[string]interface{}{
				"table":     entry.table,
				"key":       entry.key,
				"hook_name": hookName,
			}, "Deleting entities from '%s' table with key %v", entry.table, entry.key)
			tx.Table(entry.table).Where(entry.keyname+" = ?", entry.key).Delete("")
		}

		// Delete the work item cache as well
		// NOTE: Feel free to add more cache freeing calls here as needed.
		workitem.ClearGlobalWorkItemTypeCache()

		if !inTransaction {
			tx.Commit()
		}
	}
}
