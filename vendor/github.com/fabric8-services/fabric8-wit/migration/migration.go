package migration

import (
	"bufio"
	"bytes"
	"database/sql"
	"net/http"
	"net/url"
	"sync"
	"text/template"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"context"

	"github.com/goadesign/goa"
	"github.com/goadesign/goa/client"
	"github.com/jinzhu/gorm"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// AdvisoryLockID is a random number that should be used within the application
// by anybody who wants to modify the "version" table.
const AdvisoryLockID = 42

// fn defines the type of function that can be part of a migration steps
type fn func(tx *sql.Tx) error

// steps defines a collection of all the functions that make up a version
type steps []fn

// Migrations defines all a collection of all the steps
type Migrations []steps

// mutex variable to lock/unlock the population of common types
var populateLocker = &sync.Mutex{}

// Migrate executes the required migration of the database on startup.
// For each successful migration, an entry will be written into the "version"
// table, that states when a certain version was reached.
func Migrate(db *sql.DB, catalog string) error {
	var err error

	if db == nil {
		return errs.Errorf("Database handle is nil\n")
	}

	m := GetMigrations()

	var tx *sql.Tx
	for nextVersion := int64(0); nextVersion < int64(len(m)) && err == nil; nextVersion++ {

		tx, err = db.Begin()
		if err != nil {
			return errs.Errorf("Failed to start transaction: %s\n", err)
		}

		err = MigrateToNextVersion(tx, &nextVersion, m, catalog)

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

// GetMigrations returns the migrations all the migrations we have.
// Add your own migration to the end of this function.
// IMPORTANT: ALWAYS APPEND AT THE END AND DON'T CHANGE THE ORDER OF MIGRATIONS!
func GetMigrations() Migrations {
	m := Migrations{}

	// Version 0
	m = append(m, steps{ExecuteSQLFile("000-bootstrap.sql")})

	// Version 1
	m = append(m, steps{ExecuteSQLFile("001-common.sql")})

	// Version 2
	m = append(m, steps{ExecuteSQLFile("002-tracker-items.sql")})

	// Version 3
	m = append(m, steps{ExecuteSQLFile("003-login.sql")})

	// Version 4
	m = append(m, steps{ExecuteSQLFile("004-drop-tracker-query-id.sql")})

	// Version 5
	m = append(m, steps{ExecuteSQLFile("005-add-search-index.sql")})

	// Version 6
	m = append(m, steps{ExecuteSQLFile("006-rename-parent-path.sql")})

	// Version 7
	m = append(m, steps{ExecuteSQLFile("007-work-item-links.sql")})

	// Version 8
	m = append(m, steps{ExecuteSQLFile("008-soft-delete-or-resurrect.sql")})

	// Version 9
	m = append(m, steps{ExecuteSQLFile("009-drop-wit-trigger.sql")})

	// Version 10
	m = append(m, steps{ExecuteSQLFile("010-comments.sql")})

	// Version 11
	m = append(m, steps{ExecuteSQLFile("011-projects.sql")})

	// Version 12
	m = append(m, steps{ExecuteSQLFile("012-unique-work-item-links.sql")})

	// version 13
	m = append(m, steps{ExecuteSQLFile("013-iterations.sql")})

	// Version 14
	m = append(m, steps{ExecuteSQLFile("014-wi-fields-index.sql")})

	// Version 15
	m = append(m, steps{ExecuteSQLFile("015-rename-projects-to-spaces.sql")})

	// Version 16
	m = append(m, steps{ExecuteSQLFile("016-drop-wi-links-trigger.sql")})

	// Version 17
	m = append(m, steps{ExecuteSQLFile("017-alter-iterations.sql")})

	// Version 18
	m = append(m, steps{ExecuteSQLFile("018-rewrite-wits.sql")})

	// Version 19
	m = append(m, steps{ExecuteSQLFile("019-add-state-iterations.sql")})

	// Version 20
	m = append(m, steps{ExecuteSQLFile("020-work-item-description-update-search-index.sql")})

	// Version 21
	m = append(m, steps{ExecuteSQLFile("021-add-space-description.sql")})

	// Version 22
	m = append(m, steps{ExecuteSQLFile("022-work-item-description-update.sql")})

	// Version 23
	m = append(m, steps{ExecuteSQLFile("023-comment-markup.sql")})

	// Version 24
	m = append(m, steps{ExecuteSQLFile("024-comment-markup-default.sql")})

	// Version 25
	m = append(m, steps{ExecuteSQLFile("025-refactor-identities-users.sql")})

	// version 26
	m = append(m, steps{ExecuteSQLFile("026-areas.sql")})

	// version 27
	m = append(m, steps{ExecuteSQLFile("027-areas-index.sql")})

	// Version 28
	m = append(m, steps{ExecuteSQLFile("028-identity-provider_url.sql")})

	// Version 29
	m = append(m, steps{ExecuteSQLFile("029-identities-foreign-key.sql")})

	// Version 30
	m = append(m, steps{ExecuteSQLFile("030-indentities-unique-index.sql")})

	// Version 31
	m = append(m, steps{ExecuteSQLFile("031-iterations-parent-path-ltree.sql")})

	// Version 32
	m = append(m, steps{ExecuteSQLFile("032-add-foreign-key-space-id.sql")})

	// Version 33
	m = append(m, steps{ExecuteSQLFile("033-add-space-id-wilt.sql", space.SystemSpace.String(), "system.space", "Description of the space")})

	// Version 34
	m = append(m, steps{ExecuteSQLFile("034-space-owner.sql")})

	// Version 35
	m = append(m, steps{ExecuteSQLFile("035-wit-to-use-uuid.sql",
		workitem.SystemPlannerItem.String(),
		workitem.SystemTask.String(),
		workitem.SystemValueProposition.String(),
		workitem.SystemFundamental.String(),
		workitem.SystemExperience.String(),
		workitem.SystemFeature.String(),
		workitem.SystemScenario.String(),
		workitem.SystemBug.String())})

	// Version 36
	m = append(m, steps{ExecuteSQLFile("036-add-icon-to-wit.sql")})

	// version 37
	m = append(m, steps{ExecuteSQLFile("037-work-item-revisions.sql")})

	// Version 38
	m = append(m, steps{ExecuteSQLFile("038-comment-revisions.sql")})

	// Version 39
	m = append(m, steps{ExecuteSQLFile("039-comment-revisions-parentid.sql")})

	// Version 40
	m = append(m, steps{ExecuteSQLFile("040-add-space-id-wi-wit-tq.sql", space.SystemSpace.String())})

	// version 41
	m = append(m, steps{ExecuteSQLFile("041-unique-area-name-create-new-area.sql")})

	// Version 42
	m = append(m, steps{ExecuteSQLFile("042-work-item-link-revisions.sql")})

	// Version 43
	m = append(m, steps{ExecuteSQLFile("043-space-resources.sql")})

	// Version 44
	m = append(m, steps{ExecuteSQLFile("044-add-contextinfo-column-users.sql")})

	// Version 45
	m = append(m, steps{ExecuteSQLFile("045-adds-order-to-existing-wi.sql")})

	// Version 46
	m = append(m, steps{ExecuteSQLFile("046-oauth-states.sql")})

	// Version 47
	m = append(m, steps{ExecuteSQLFile("047-codebases.sql")})

	// Version 48
	m = append(m, steps{ExecuteSQLFile("048-unique-iteration-name-create-new-iteration.sql")})

	// Version 49
	m = append(m, steps{ExecuteSQLFile("049-add-wi-to-root-area.sql")})

	// Version 50
	m = append(m, steps{ExecuteSQLFile("050-add-company-to-user-profile.sql")})

	// Version 51
	m = append(m, steps{ExecuteSQLFile("051-modify-work_item_link_types_name_idx.sql")})

	// Version 52
	m = append(m, steps{ExecuteSQLFile("052-unique-space-names.sql")})

	// Version 53
	m = append(m, steps{ExecuteSQLFile("053-edit-username.sql")})

	// Version 54
	m = append(m, steps{ExecuteSQLFile("054-add-stackid-to-codebase.sql")})

	// Version 55
	m = append(m, steps{ExecuteSQLFile("055-assign-root-area-if-missing.sql")})

	// Version 56
	m = append(m, steps{ExecuteSQLFile("056-assign-root-iteration-if-missing.sql")})

	// Version 57
	m = append(m, steps{ExecuteSQLFile("057-add-last-used-workspace-to-codebase.sql")})

	// Version 58
	m = append(m, steps{ExecuteSQLFile("058-index-identities-fullname.sql")})

	// Version 59
	m = append(m, steps{ExecuteSQLFile("059-fixed-ids-for-system-link-types-and-categories.sql",
		link.SystemWorkItemLinkTypeBugBlockerID.String(),
		link.SystemWorkItemLinkPlannerItemRelatedID.String(),
		link.SystemWorkItemLinkTypeParentChildID.String(),
		link.SystemWorkItemLinkCategorySystemID.String(),
		link.SystemWorkItemLinkCategoryUserID.String())})

	// Version 60
	m = append(m, steps{ExecuteSQLFile("060-fixed-identities-username-idx.sql")})

	// Version 61
	m = append(m, steps{ExecuteSQLFile("061-replace-index-space-name.sql")})

	// Version 62
	m = append(m, steps{ExecuteSQLFile("062-link-system-preparation.sql")})

	// Version 63
	m = append(m, steps{ExecuteSQLFile("063-workitem-related-changes.sql")})

	// Version 64
	m = append(m, steps{ExecuteSQLFile("064-remove-link-combinations.sql")})

	// Version 65
	m = append(m, steps{ExecuteSQLFile("065-workitem-id-unique-per-space.sql")})

	// Version 66
	m = append(m, steps{ExecuteSQLFile("066-work_item_links_data_integrity.sql")})

	// Version 67
	m = append(m, steps{ExecuteSQLFile("067-comment-parentid-uuid.sql")})

	// Version 68
	m = append(m, steps{ExecuteSQLFile("068-index_identities_username.sql")})

	// Version 69
	m = append(m, steps{ExecuteSQLFile("069-limit-execution-order-to-space.sql")})

	// Version 70
	m = append(m, steps{ExecuteSQLFile("070-rename-comment-createdby-to-creator.sql")})

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
			ExecuteSQLFile("YOUR_OWN_FILE.sql"),
			func(db *sql.Tx) error {
				// Execute random go code
				return nil
			},
		})
	*/

	return m
}

// ExecuteSQLFile loads the given filename from the packaged SQL files and
// executes it on the given database. Golang text/template module is used
// to handle all the optional arguments passed to the sql files
func ExecuteSQLFile(filename string, args ...string) fn {
	return func(db *sql.Tx) error {
		data, err := Asset(filename)
		if err != nil {
			return errs.Wrapf(err, "failed to find filename: %s", filename)
		}

		if len(args) > 0 {
			tmpl, err := template.New("sql").Parse(string(data))
			if err != nil {
				return errs.Wrap(err, "failed to parse SQL template")
			}
			var sqlScript bytes.Buffer
			writer := bufio.NewWriter(&sqlScript)
			err = tmpl.Execute(writer, args)
			if err != nil {
				return errs.Wrap(err, "failed to execute SQL template")
			}
			// We need to flush the content of the writer
			writer.Flush()
			_, err = db.Exec(sqlScript.String())
			if err != nil {
				log.Error(context.Background(), map[string]interface{}{}, "failed to execute this query: \n\n%s\n\n", sqlScript.String())
			}
		} else {
			_, err = db.Exec(string(data))
			if err != nil {
				log.Error(context.Background(), map[string]interface{}{}, "failed to execute this query: \n\n%s\n\n", string(data))
			}
		}

		return errs.WithStack(err)
	}
}

// MigrateToNextVersion migrates the database to the nextVersion.
// If the database is already at nextVersion or higher, the nextVersion
// will be set to the actual next version.
func MigrateToNextVersion(tx *sql.Tx, nextVersion *int64, m Migrations, catalog string) error {
	// Obtain exclusive transaction level advisory that doesn't depend on any table.
	// Once obtained, the lock is held for the remainder of the current transaction.
	// (There is no UNLOCK TABLE command; locks are always released at transaction end.)
	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", AdvisoryLockID); err != nil {
		return errs.Errorf("Failed to acquire lock: %s\n", err)
	}

	// Determine current version and adjust the outmost loop
	// iterator variable "version"
	currentVersion, err := getCurrentVersion(tx, catalog)
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
func getCurrentVersion(db *sql.Tx, catalog string) (int64, error) {
	query := `SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_catalog=$1
				AND table_name='version')`
	row := db.QueryRow(query, catalog)

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
	var reqID string
	ctx, reqID = client.ContextWithRequestID(ctx)

	log.Debug(ctx, nil, "Initialized the migration context with Request ID: %v", reqID)

	return ctx
}

// BootstrapWorkItemLinking makes sure the database is populated with the correct work item link stuff (e.g. category and some basic types)
func BootstrapWorkItemLinking(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, spaceRepo *space.GormRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository) error {
	populateLocker.Lock()
	defer populateLocker.Unlock()
	if err := createOrUpdateSpace(ctx, spaceRepo, space.SystemSpace, "The system space is reserved for spaces that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	// create link categories
	systemCatDesc := "The system category is reserved for link types that are to be manipulated by the system only."
	systemCat := link.WorkItemLinkCategory{
		ID:          link.SystemWorkItemLinkCategorySystemID,
		Name:        "system",
		Description: &systemCatDesc,
	}
	_, err := createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, &systemCat)
	if err != nil {
		return errs.WithStack(err)
	}
	userCatDesc := "The user category is reserved for link types that can to be manipulated by the user."
	userCat := link.WorkItemLinkCategory{
		ID:          link.SystemWorkItemLinkCategoryUserID,
		Name:        "user",
		Description: &userCatDesc,
	}
	_, err = createOrUpdateWorkItemLinkCategory(ctx, linkCatRepo, &userCat)
	if err != nil {
		return errs.WithStack(err)
	}

	// create work item link types
	blockerDesc := "One bug blocks a planner item."
	blockerWILT := link.WorkItemLinkType{
		ID:             link.SystemWorkItemLinkTypeBugBlockerID,
		Name:           "Bug blocker",
		Description:    &blockerDesc,
		Topology:       link.TopologyNetwork,
		ForwardName:    "blocks",
		ReverseName:    "blocked by",
		LinkCategoryID: systemCat.ID,
		SpaceID:        space.SystemSpace,
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, &blockerWILT); err != nil {
		return errs.WithStack(err)
	}
	relatedDesc := "One planner item or a subtype of it relates to another one."
	relatedWILT := link.WorkItemLinkType{
		ID:             link.SystemWorkItemLinkPlannerItemRelatedID,
		Name:           "Related planner item",
		Description:    &relatedDesc,
		Topology:       link.TopologyNetwork,
		ForwardName:    "relates to",
		ReverseName:    "is related to",
		LinkCategoryID: systemCat.ID,
		SpaceID:        space.SystemSpace,
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, &relatedWILT); err != nil {
		return errs.WithStack(err)
	}
	parentingDesc := "One planner item or a subtype of it which is a parent of another one."
	parentingWILT := link.WorkItemLinkType{
		ID:             link.SystemWorkItemLinkTypeParentChildID,
		Name:           "Parent child item",
		Description:    &parentingDesc,
		Topology:       link.TopologyNetwork,
		ForwardName:    "parent of",
		ReverseName:    "child of",
		LinkCategoryID: systemCat.ID,
		SpaceID:        space.SystemSpace,
	}
	if err := createOrUpdateWorkItemLinkType(ctx, linkCatRepo, linkTypeRepo, spaceRepo, &parentingWILT); err != nil {
		return errs.WithStack(err)
	}
	return nil
}

func createOrUpdateWorkItemLinkCategory(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, linkCat *link.WorkItemLinkCategory) (*link.WorkItemLinkCategory, error) {
	cat, err := linkCatRepo.Load(ctx, linkCat.ID)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		cat, err = linkCatRepo.Create(ctx, linkCat)
		if err != nil {
			return nil, errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"category": linkCat,
		}, "Work item link category %s exists, will update/overwrite the description", linkCat.Name)

		cat.Description = linkCat.Description
		cat, err = linkCatRepo.Save(ctx, *cat)
		if err != nil {
			return nil, errs.WithStack(err)
		}
	}
	return cat, nil
}

func createOrUpdateSpace(ctx context.Context, spaceRepo *space.GormRepository, id uuid.UUID, description string) error {
	s, err := spaceRepo.Load(ctx, id)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"space_id": id,
		}, "space %s will be created", id)
		newSpace := &space.Space{
			Description: description,
			Name:        "system.space",
			ID:          id,
		}
		_, err := spaceRepo.Create(ctx, newSpace)
		if err != nil {
			return errs.Wrapf(err, "failed to create space %s", id)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"pkg":      "migration",
			"space_id": id,
		}, "space %s exists, will update/overwrite the description", id)

		s.Description = description
		_, err = spaceRepo.Save(ctx, s)
		return errs.WithStack(err)
	}
	return nil
}

func createSpace(ctx context.Context, spaceRepo *space.GormRepository, id uuid.UUID, description string) error {
	err := spaceRepo.CheckExists(ctx, id.String())
	if err != nil {
		cause := errs.Cause(err)
		switch cause.(type) {
		case errors.NotFoundError:
			log.Info(ctx, map[string]interface{}{
				"pkg":      "migration",
				"space_id": id,
			}, "space %s will be created", id)
			newSpace := &space.Space{
				Description: description,
				Name:        "system.space",
				ID:          id,
			}
			_, err := spaceRepo.Create(ctx, newSpace)
			if err != nil {
				return errs.Wrapf(err, "failed to create space %s", id)
			}
		default:
			log.Error(ctx, map[string]interface{}{"err": err, "space_id": id}, "unable to verify if a space exists")
		}
	}
	return nil
}

func createOrUpdateWorkItemLinkType(ctx context.Context, linkCatRepo *link.GormWorkItemLinkCategoryRepository, linkTypeRepo *link.GormWorkItemLinkTypeRepository, spaceRepo *space.GormRepository, linkType *link.WorkItemLinkType) error {
	existingLinkType, err := linkTypeRepo.Load(ctx, linkType.ID)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := linkTypeRepo.Create(ctx, linkType)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"wilt": linkType.Name,
		}, "Work item link type %s exists, will update/overwrite all fields", linkType.Name)
		linkType.ID = existingLinkType.ID
		linkType.Version = existingLinkType.Version
		_, err = linkTypeRepo.Save(ctx, *linkType)
		return errs.WithStack(err)
	}
	return nil
}

// PopulateCommonTypes makes sure the database is populated with the correct types (e.g. bug etc.)
func PopulateCommonTypes(ctx context.Context, db *gorm.DB, witr *workitem.GormWorkItemTypeRepository) error {
	populateLocker.Lock()
	defer populateLocker.Unlock()
	if err := createSpace(ctx, space.NewRepository(db), space.SystemSpace, "The system space is reserved for spaces that can to be manipulated by the user."); err != nil {
		return errs.WithStack(err)
	}
	if err := createOrUpdateSystemPlannerItemType(ctx, witr, db, space.SystemSpace); err != nil {
		return errs.WithStack(err)
	}
	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs

	type witInfo struct {
		id          uuid.UUID
		name        string
		description string
		icon        string
	}

	info := []witInfo{
		{workitem.SystemBug, "Bug", "", "fa fa-bug"},
		{workitem.SystemTask, "Task", "", "fa fa-tasks"},
		{workitem.SystemFeature, "Feature", "", "fa fa-puzzle-piece"},
		{workitem.SystemScenario, "Scenario", "", "fa fa-bolt"},
		{workitem.SystemValueProposition, "Value Proposition", "", "fa fa-diamond"},
		{workitem.SystemExperience, "Experience", "", "fa fa-map"},
		{workitem.SystemFundamental, "Fundamental", "", "fa fa-university"},
		{workitem.SystemPapercuts, "Papercuts", "", "fa fa-scissors"},
	}
	for _, i := range info {
		if err := createOrUpdatePlannerItemExtension(ctx, i.id, i.name, i.description, i.icon, witr, db, space.SystemSpace); err != nil {
			return errs.Wrapf(err, "failed to create WIT with %+v", i)
		}
	}

	workitem.ClearGlobalWorkItemTypeCache() // Clear the WIT cache after updating existing WITs
	return nil
}

func createOrUpdateSystemPlannerItemType(ctx context.Context, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB, spaceID uuid.UUID) error {
	log.Info(ctx, nil, "Creating or updating planner item type...")
	typeID := workitem.SystemPlannerItem
	typeName := "Planner Item"
	description := "Description for Planner Item"
	icon := "fa fa-bookmark"
	workItemTypeFields := map[string]workitem.FieldDefinition{
		workitem.SystemTitle:        {Type: workitem.SimpleType{Kind: "string"}, Required: true, Label: "Title", Description: "The title text of the work item"},
		workitem.SystemDescription:  {Type: workitem.SimpleType{Kind: "markup"}, Required: false, Label: "Description", Description: "A descriptive text of the work item"},
		workitem.SystemCreator:      {Type: workitem.SimpleType{Kind: "user"}, Required: true, Label: "Creator", Description: "The user that created the work item"},
		workitem.SystemRemoteItemID: {Type: workitem.SimpleType{Kind: "string"}, Required: false, Label: "Remote item", Description: "The ID of the remote work item"},
		workitem.SystemCreatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Created at", Description: "The date and time when the work item was created"},
		workitem.SystemUpdatedAt:    {Type: workitem.SimpleType{Kind: "instant"}, Required: false, Label: "Updated at", Description: "The date and time when the work item was last updated"},
		workitem.SystemOrder:        {Type: workitem.SimpleType{Kind: "float"}, Required: false, Label: "Execution Order", Description: "Execution Order of the workitem."},
		workitem.SystemIteration:    {Type: workitem.SimpleType{Kind: "iteration"}, Required: false, Label: "Iteration", Description: "The iteration to which the work item belongs"},
		workitem.SystemArea:         {Type: workitem.SimpleType{Kind: "area"}, Required: false, Label: "Area", Description: "The area to which the work item belongs"},
		workitem.SystemCodebase:     {Type: workitem.SimpleType{Kind: "codebase"}, Required: false, Label: "Codebase", Description: "Contains codebase attributes to which this WI belongs to"},
		workitem.SystemAssignees: {
			Type: &workitem.ListType{
				SimpleType:    workitem.SimpleType{Kind: workitem.KindList},
				ComponentType: workitem.SimpleType{Kind: workitem.KindUser}},
			Required:    false,
			Label:       "Assignees",
			Description: "The users that are assigned to the work item",
		},
		workitem.SystemState: {
			Type: &workitem.EnumType{
				SimpleType: workitem.SimpleType{Kind: workitem.KindEnum},
				BaseType:   workitem.SimpleType{Kind: workitem.KindString},
				Values: []interface{}{
					workitem.SystemStateNew,
					workitem.SystemStateOpen,
					workitem.SystemStateInProgress,
					workitem.SystemStateResolved,
					workitem.SystemStateClosed,
				},
			},

			Required:    true,
			Label:       "State",
			Description: "The state of the work item",
		},
	}
	return createOrUpdateType(ctx, typeID, spaceID, typeName, description, nil, workItemTypeFields, icon, witr, db)
}

func createOrUpdatePlannerItemExtension(ctx context.Context, typeID uuid.UUID, name string, description string, icon string, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB, spaceID uuid.UUID) error {
	workItemTypeFields := map[string]workitem.FieldDefinition{}
	extTypeName := workitem.SystemPlannerItem
	return createOrUpdateType(ctx, typeID, spaceID, name, description, &extTypeName, workItemTypeFields, icon, witr, db)
}

func createOrUpdateType(ctx context.Context, typeID uuid.UUID, spaceID uuid.UUID, name string, description string, extendedTypeID *uuid.UUID, fields map[string]workitem.FieldDefinition, icon string, witr *workitem.GormWorkItemTypeRepository, db *gorm.DB) error {
	log.Info(ctx, nil, "Creating or updating planner item types...")
	wit, err := witr.LoadTypeFromDB(ctx, typeID)
	cause := errs.Cause(err)
	switch cause.(type) {
	case errors.NotFoundError:
		_, err := witr.Create(ctx, spaceID, &typeID, extendedTypeID, name, &description, icon, fields)
		if err != nil {
			return errs.WithStack(err)
		}
	case nil:
		log.Info(ctx, map[string]interface{}{
			"type_id": typeID,
		}, "Work item type %s exists, will update/overwrite the fields, name, icon, description and parentPath", typeID.String())

		path := workitem.LtreeSafeID(typeID)
		if extendedTypeID != nil {
			log.Info(ctx, map[string]interface{}{
				"type_id":          typeID,
				"extended_type_id": *extendedTypeID,
			}, "Work item type %v extends another type %v will copy fields from the extended type", typeID, *extendedTypeID)

			extendedWit, err := witr.LoadTypeFromDB(ctx, *extendedTypeID)
			if err != nil {
				return errs.WithStack(err)
			}
			path = extendedWit.Path + workitem.GetTypePathSeparator() + path

			//load fields from the extended type
			err = loadFields(ctx, extendedWit, fields)
			if err != nil {
				return errs.WithStack(err)
			}
		}

		if err != nil {
			return errs.WithStack(err)
		}
		wit.Name = name
		wit.Description = &description
		wit.Icon = icon
		wit.Fields = fields
		wit.Path = path
		db = db.Save(wit)
		return db.Error
	}
	log.Info(ctx, nil, "Creating or updating planner item type done.")
	return nil
}

func loadFields(ctx context.Context, wit *workitem.WorkItemType, into workitem.FieldDefinitions) error {
	// copy fields from wit
	for key, value := range wit.Fields {
		// do not overwrite already defined fields in the map
		if _, exist := into[key]; !exist {
			into[key] = value
		} else {
			// If field already exist, overwrite only the label and description
			into[key] = workitem.FieldDefinition{
				Label:       value.Label,
				Description: value.Description,
				Required:    into[key].Required,
				Type:        into[key].Type,
			}
		}
	}

	return nil
}
