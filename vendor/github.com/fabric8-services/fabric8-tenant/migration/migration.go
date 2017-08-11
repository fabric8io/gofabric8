package migration

import (
	"bufio"
	"bytes"
	"database/sql"
	"net/http"
	"net/url"
	"sync"
	"text/template"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	errs "github.com/pkg/errors"
	"golang.org/x/net/context"
)

// AdvisoryLockID is a random number that should be used within the application
// by anybody who wants to modify the "version" table.
const AdvisoryLockID = 42

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

// steps defines a collection of all the functions that make up a version
type steps []fn

// migrations defines all a collection of all the steps
type migrations []steps

// mutex variable to lock/unlock the population of common types
var populateLocker = &sync.Mutex{}

// Migrate executes the required migration of the database on startup.
// For each successful migration, an entry will be written into the "version"
// table, that states when a certain version was reached.
func Migrate(db *sql.DB) error {
	var err error

	if db == nil {
		return errs.Errorf("Database handle is nil\n")
	}

	m := getMigrations()

	var tx *sql.Tx
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {

		tx, err = db.Begin()
		if err != nil {
			return errs.Errorf("Failed to start transaction: %s\n", err)
		}

		err = migrateToNextVersion(tx, &nextVersion, m)

		if err != nil {
			oldErr := err
			log.Info(nil, map[string]interface{}{
				"next_version": nextVersion,
				"migrations":   m,
				"err":          err,
			}, "Rolling back transaction due to: %v", err)

			if err = tx.Rollback(); err != nil {
				log.Error(nil, map[string]interface{}{
					"next_version": nextVersion,
					"migrations":   m,
					"err":          err,
				}, "error while rolling back transaction: ", err)
				return errs.Errorf("Error while rolling back transaction: %s\n", err)
			}
			return oldErr
		}

		if err = tx.Commit(); err != nil {
			log.Error(nil, map[string]interface{}{
				"migrations": m,
				"err":        err,
			}, "error during transaction commit: %v", err)
			return errs.Errorf("Error during transaction commit: %s\n", err)
		}

	}

	if err != nil {
		log.Error(nil, map[string]interface{}{
			"migrations": m,
			"err":        err,
		}, "migration failed with error: %v", err)
		return errs.Errorf("Migration failed with error: %s\n", err)
	}

	return nil
}

// getMigrations returns the migrations all the migrations we have.
// Add your own migration to the end of this function.
// IMPORTANT: ALWAYS APPEND AT THE END AND DON'T CHANGE THE ORDER OF MIGRATIONS!
func getMigrations() migrations {
	m := migrations{}

	m = append(m, steps{executeSQLFile("000-bootstrap.sql")})
	m = append(m, steps{executeSQLFile("001-tenant-and-namespaces.sql")})

	// Version N
	//
	// In order to add an upgrade, simply append an array of MigrationFunc to the
	// the end of the "migrations" slice. The version numbers are determined by
	// the index in the array. The following code in comments show how you can
	// do a migration in 3 steps. If one of the steps fails, the others are not
	// executed.
	// If something goes wrong during the migration, all you need to do is return
	// an error that is not nil.

	/*
		m = append(m, steps{
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
			executeSQLFile("YOUR_OWN_FILE.sql"),
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
		})
	*/

	return m
}

// executeSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql files
func executeSQLFile(filename string, args ...string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return errs.WithStack(err)
		}

		if len(args) > 0 {
			tmpl, err := template.New("sql").Parse(string(data))
			if err != nil {
				return errs.WithStack(err)
			}
			var sqlScript bytes.Buffer
			writer := bufio.NewWriter(&sqlScript)
			err = tmpl.Execute(writer, args)
			if err != nil {
				return errs.WithStack(err)
			}
			// We need to flush the content of the writer
			writer.Flush()
			_, err = db.Exec(sqlScript.String())
		} else {
			_, err = db.Exec(string(data))
		}

		return errs.WithStack(err)
	}
}

// migrateToNextVersion migrates the database to the nextVersion.
// If the database is already at nextVersion or higher, the nextVersion
// will be set to the actual next version.
func migrateToNextVersion(tx *sql.Tx, nextVersion *int64, m migrations) error {
	// Obtain exclusive transaction level advisory that doesn't depend on any table.
	// Once obtained, the lock is held for the remainder of the current transaction.
	// (There is no UNLOCK TABLE command; locks are always released at transaction end.)
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", AdvisoryLockID); err != nil {
		return errs.Errorf("Failed to acquire lock: %s\n", err)
	}

	// Determine current version and adjust the outmost loop
	// iterator variable "version"
	currentVersion, err := getCurrentVersion(tx)
	if err != nil {
		return errs.WithStack(err)
	}
	*nextVersion = currentVersion + 1
	if *nextVersion >= int64(len(m)) {
		// No further updates to apply (this is NOT an error)
		log.Info(nil, map[string]interface{}{
			"next_version":    *nextVersion,
			"current_version": currentVersion,
		}, "Current version %d. Nothing to update.", currentVersion)
		return nil
	}

	log.Info(nil, map[string]interface{}{
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Attempt to update DB to version %v", *nextVersion)

	// Apply all the updates of the next version
	for j := range m[*nextVersion] {
		if err := m[*nextVersion][j](tx); err != nil {
			return errs.Errorf("Failed to execute migration of step %d of version %d: %s\n", j, *nextVersion, err)
		}
	}

	if _, err := tx.Exec("INSERT INTO version(version) VALUES($1)", *nextVersion); err != nil {
		return errs.Errorf("Failed to update DB to version %d: %s\n", *nextVersion, err)
	}

	log.Info(nil, map[string]interface{}{
		"next_version":    *nextVersion,
		"current_version": currentVersion,
	}, "Successfully updated DB to version %v", *nextVersion)

	return nil
}

// getCurrentVersion returns the highest version from the version
// table or -1 if that table does not exist.
//
// Returning -1 simplifies the logic of the migration process because
// the next version is always the current version + 1 which results
// in -1 + 1 = 0 which is exactly what we want as the first version.
func getCurrentVersion(db *sql.Tx) (int64, error) {
	row := db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='version')")

	var exists bool
	if err := row.Scan(&exists); err != nil {
		return -1, errs.Errorf("Failed to scan if table \"version\" exists: %s\n", err)
	}

	if !exists {
		// table doesn't exist
		return -1, nil
	}

	row = db.QueryRow("SELECT max(version) as current FROM version")

	var current int64 = -1
	if err := row.Scan(&current); err != nil {
		return -1, errs.Errorf("Failed to scan max version in table \"version\": %s\n", err)
	}

	return current, nil
}

// NewMigrationContext aims to create a new goa context where to initialize the
// request and req_id context keys.
// NOTE: We need this function to initialize the goa.ContextRequest
func NewMigrationContext(ctx context.Context) context.Context {
	req := &http.Request{Host: "localhost"}
	params := url.Values{}
	ctx = goa.NewContext(ctx, nil, req, params)
	// set a random request ID for the context
	var req_id string
	ctx, req_id = client.ContextWithRequestID(ctx)

	log.Debug(ctx, nil, "Initialized the migration context with Request ID: %v", req_id)

	return ctx
}
