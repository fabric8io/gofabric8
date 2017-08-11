package main

import (
	"flag"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

	"context"

	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/auth"
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/controller"
	witmiddleware "github.com/fabric8-services/fabric8-wit/goamiddleware"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/remoteworkitem"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/fabric8-services/fabric8-wit/workitem/link"

	"github.com/goadesign/goa"
	goalogrus "github.com/goadesign/goa/logging/logrus"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
)

func main() {
	// --------------------------------------------------------------------
	// Parse flags
	// --------------------------------------------------------------------
	var configFilePath string
	var printConfig bool
	var migrateDB bool
	var scheduler *remoteworkitem.Scheduler
	flag.StringVar(&configFilePath, "config", "", "Path to the config file to read")
	flag.BoolVar(&printConfig, "printConfig", false, "Prints the config (including merged environment variables) and exits")
	flag.BoolVar(&migrateDB, "migrateDatabase", false, "Migrates the database to the newest version and exits.")
	flag.Parse()

	// Override default -config switch with environment variable only if -config switch was
	// not explicitly given via the command line.
	configSwitchIsSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			configSwitchIsSet = true
		}
	})
	if !configSwitchIsSet {
		if envConfigPath, ok := os.LookupEnv("F8_CONFIG_FILE_PATH"); ok {
			configFilePath = envConfigPath
		}
	}

	configuration, err := config.NewConfigurationData(configFilePath)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"config_file_path": configFilePath,
			"err":              err,
		}, "failed to setup the configuration")
	}

	if printConfig {
		os.Exit(0)
	}

	// Initialized developer mode flag and log level for the logger
	log.InitializeLogger(configuration.IsLogJSON(), configuration.GetLogLevel())

	printUserInfo()

	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", configuration.GetPostgresConfigString())
		if err != nil {
			db.Close()
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", configuration.GetPostgresConnectionRetrySleep())
			time.Sleep(configuration.GetPostgresConnectionRetrySleep())
		} else {
			defer db.Close()
			break
		}
	}

	if configuration.IsPostgresDeveloperModeEnabled() && log.IsDebug() {
		db = db.Debug()
	}

	if configuration.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", configuration.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(configuration.GetPostgresConnectionMaxIdle())
	}
	if configuration.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", configuration.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(configuration.GetPostgresConnectionMaxOpen())
	}

	// Set the database transaction timeout
	application.SetDatabaseTransactionTimeout(configuration.GetPostgresTransactionTimeout())

	// Migrate the schema
	err = migration.Migrate(db.DB(), configuration.GetPostgresDatabase())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed migration")
	}

	// Nothing to here except exit, since the migration is already performed.
	if migrateDB {
		os.Exit(0)
	}

	// Make sure the database is populated with the correct types (e.g. bug etc.)
	if configuration.GetPopulateCommonTypes() {
		ctx := migration.NewMigrationContext(context.Background())

		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to populate common types")
		}
		if err := models.Transactional(db, func(tx *gorm.DB) error {
			return migration.BootstrapWorkItemLinking(ctx, link.NewWorkItemLinkCategoryRepository(tx), space.NewRepository(tx), link.NewWorkItemLinkTypeRepository(tx))
		}); err != nil {
			log.Panic(ctx, map[string]interface{}{
				"err": err,
			}, "failed to bootstap work item linking")
		}
	}

	// Create service
	service := goa.New("wit")

	// Mount middleware
	service.Use(middleware.RequestID())
	// Use our own log request to inject identity id and modify other properties
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())

	service.WithLogger(goalogrus.New(log.Logger()))

	// Setup Account/Login/Security
	identityRepository := account.NewIdentityRepository(db)
	userRepository := account.NewUserRepository(db)

	var notificationChannel notification.Channel = &notification.DevNullChannel{}
	if configuration.GetNotificationServiceURL() != "" {
		log.Logger().Infof("Enabling Notification service %v", configuration.GetNotificationServiceURL())
		channel, err := notification.NewServiceChannel(configuration)
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err": err,
				"url": configuration.GetNotificationServiceURL(),
			}, "failed to parse notification service url")
		}
		notificationChannel = channel
	}

	appDB := gormapplication.NewGormDB(db)

	publicKey, err := token.ParsePublicKey(configuration.GetTokenPublicKey())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to parse public token")
	}
	tokenManager := token.NewManager(publicKey)
	// Middleware that extracts and stores the token in the context
	jwtMiddlewareTokenContext := witmiddleware.TokenContext(publicKey, nil, app.NewJWTSecurity())
	service.Use(jwtMiddlewareTokenContext)

	service.Use(login.InjectTokenManager(tokenManager))
	service.Use(log.LogRequest(configuration.IsPostgresDeveloperModeEnabled()))
	app.UseJWTMiddleware(service, goajwt.New(publicKey, nil, app.NewJWTSecurity()))

	spaceAuthzService := authz.NewAuthzService(configuration, appDB)
	service.Use(authz.InjectAuthzService(spaceAuthzService))

	loginService := login.NewKeycloakOAuthProvider(identityRepository, userRepository, tokenManager, appDB)
	loginCtrl := controller.NewLoginController(service, loginService, tokenManager, configuration, identityRepository)
	app.MountLoginController(service, loginCtrl)

	logoutCtrl := controller.NewLogoutController(service, &login.KeycloakLogoutService{}, configuration)
	app.MountLogoutController(service, logoutCtrl)

	// Mount "status" controller
	statusCtrl := controller.NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	// Mount "workitem" controller
	//workitemCtrl := controller.NewWorkitemController(service, appDB, configuration)
	workitemCtrl := controller.NewNotifyingWorkitemController(service, appDB, notificationChannel, configuration)
	app.MountWorkitemController(service, workitemCtrl)

	// Mount "named workitem" controller
	namedWorkitemsCtrl := controller.NewNamedWorkItemsController(service, appDB)
	app.MountNamedWorkItemsController(service, namedWorkitemsCtrl)

	// Mount "workitems" controller
	//workitemsCtrl := controller.NewWorkitemsController(service, appDB, configuration)
	workitemsCtrl := controller.NewNotifyingWorkitemsController(service, appDB, notificationChannel, configuration)
	app.MountWorkitemsController(service, workitemsCtrl)

	// Mount "workitemtype" controller
	workitemtypeCtrl := controller.NewWorkitemtypeController(service, appDB, configuration)
	app.MountWorkitemtypeController(service, workitemtypeCtrl)

	// Mount "work item link category" controller
	workItemLinkCategoryCtrl := controller.NewWorkItemLinkCategoryController(service, appDB)
	app.MountWorkItemLinkCategoryController(service, workItemLinkCategoryCtrl)

	// Mount "work item link type" controller
	workItemLinkTypeCtrl := controller.NewWorkItemLinkTypeController(service, appDB, configuration)
	app.MountWorkItemLinkTypeController(service, workItemLinkTypeCtrl)

	// Mount "work item link" controller
	workItemLinkCtrl := controller.NewWorkItemLinkController(service, appDB, configuration)
	app.MountWorkItemLinkController(service, workItemLinkCtrl)

	// Mount "work item comments" controller
	//workItemCommentsCtrl := controller.NewWorkItemCommentsController(service, appDB, configuration)
	workItemCommentsCtrl := controller.NewNotifyingWorkItemCommentsController(service, appDB, notificationChannel, configuration)
	app.MountWorkItemCommentsController(service, workItemCommentsCtrl)

	// Mount "work item relationships links" controller
	workItemRelationshipsLinksCtrl := controller.NewWorkItemRelationshipsLinksController(service, appDB, configuration)
	app.MountWorkItemRelationshipsLinksController(service, workItemRelationshipsLinksCtrl)

	// Mount "comments" controller
	//commentsCtrl := controller.NewCommentsController(service, appDB, configuration)
	commentsCtrl := controller.NewNotifyingCommentsController(service, appDB, notificationChannel, configuration)
	app.MountCommentsController(service, commentsCtrl)

	if configuration.GetFeatureWorkitemRemote() {
		// Scheduler to fetch and import remote tracker items
		scheduler = remoteworkitem.NewScheduler(db)
		defer scheduler.Stop()

		accessTokens := controller.GetAccessTokens(configuration)
		scheduler.ScheduleAllQueries(service.Context, accessTokens)

		// Mount "tracker" controller
		c5 := controller.NewTrackerController(service, appDB, scheduler, configuration)
		app.MountTrackerController(service, c5)

		// Mount "trackerquery" controller
		c6 := controller.NewTrackerqueryController(service, appDB, scheduler, configuration)
		app.MountTrackerqueryController(service, c6)
	}

	// Mount "space" controller
	spaceCtrl := controller.NewSpaceController(service, appDB, configuration, auth.NewKeycloakResourceManager(configuration))
	app.MountSpaceController(service, spaceCtrl)

	// Mount "user" controller
	userCtrl := controller.NewUserController(service, appDB, tokenManager, configuration)
	if configuration.GetTenantServiceURL() != "" {
		log.Logger().Infof("Enabling Init Tenant service %v", configuration.GetTenantServiceURL())
		userCtrl.InitTenant = account.NewInitTenant(configuration)
	}
	app.MountUserController(service, userCtrl)

	userServiceCtrl := controller.NewUserServiceController(service)
	userServiceCtrl.UpdateTenant = account.NewUpdateTenant(configuration)
	app.MountUserServiceController(service, userServiceCtrl)

	// Mount "search" controller
	searchCtrl := controller.NewSearchController(service, appDB, configuration)
	app.MountSearchController(service, searchCtrl)

	// Mount "users" controller
	keycloakProfileService := login.NewKeycloakUserProfileClient()
	usersCtrl := controller.NewUsersController(service, appDB, configuration, keycloakProfileService)
	app.MountUsersController(service, usersCtrl)

	// Mount "iterations" controller
	iterationCtrl := controller.NewIterationController(service, appDB, configuration)
	app.MountIterationController(service, iterationCtrl)

	// Mount "spaceiterations" controller
	spaceIterationCtrl := controller.NewSpaceIterationsController(service, appDB, configuration)
	app.MountSpaceIterationsController(service, spaceIterationCtrl)

	// Mount "userspace" controller
	userspaceCtrl := controller.NewUserspaceController(service, db)
	app.MountUserspaceController(service, userspaceCtrl)

	// Mount "render" controller
	renderCtrl := controller.NewRenderController(service)
	app.MountRenderController(service, renderCtrl)

	// Mount "areas" controller
	areaCtrl := controller.NewAreaController(service, appDB, configuration)
	app.MountAreaController(service, areaCtrl)

	spaceAreaCtrl := controller.NewSpaceAreasController(service, appDB, configuration)
	app.MountSpaceAreasController(service, spaceAreaCtrl)

	filterCtrl := controller.NewFilterController(service, configuration)
	app.MountFilterController(service, filterCtrl)

	// Mount "namedspaces" controller
	namedSpacesCtrl := controller.NewNamedspacesController(service, appDB)
	app.MountNamedspacesController(service, namedSpacesCtrl)

	// Mount "plannerBacklog" controller
	plannerBacklogCtrl := controller.NewPlannerBacklogController(service, appDB, configuration)
	app.MountPlannerBacklogController(service, plannerBacklogCtrl)

	// Mount "codebase" controller
	codebaseCtrl := controller.NewCodebaseController(service, appDB, configuration)
	app.MountCodebaseController(service, codebaseCtrl)

	// Mount "spacecodebases" controller
	spaceCodebaseCtrl := controller.NewSpaceCodebasesController(service, appDB)
	app.MountSpaceCodebasesController(service, spaceCodebaseCtrl)

	// Mount "collaborators" controller
	collaboratorsCtrl := controller.NewCollaboratorsController(service, appDB, configuration, auth.NewKeycloakPolicyManager(configuration))
	app.MountCollaboratorsController(service, collaboratorsCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", configuration.IsPostgresDeveloperModeEnabled())
	log.Logger().Infoln("GOMAXPROCS:     ", runtime.GOMAXPROCS(-1))
	log.Logger().Infoln("NumCPU:         ", runtime.NumCPU())

	http.Handle("/api/", service.Mux)
	http.Handle("/", http.FileServer(assetFS()))
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(configuration.GetHTTPAddress(), nil); err != nil {
		log.Error(nil, map[string]interface{}{
			"addr": configuration.GetHTTPAddress(),
			"err":  err,
		}, "unable to connect to server")
		service.LogError("startup", "err", err)
	}

}

func printUserInfo() {
	u, err := user.Current()
	if err != nil {
		log.Warn(nil, map[string]interface{}{
			"err": err,
		}, "failed to get current user")
	} else {
		log.Info(nil, map[string]interface{}{
			"username": u.Username,
			"uuid":     u.Uid,
		}, "Running as user name '%s' with UID %s.", u.Username, u.Uid)
		g, err := user.LookupGroupId(u.Gid)
		if err != nil {
			log.Warn(nil, map[string]interface{}{
				"err": err,
			}, "failed to lookup group")
		} else {
			log.Info(nil, map[string]interface{}{
				"groupname": g.Name,
				"gid":       g.Gid,
			}, "Running as as group '%s' with GID %s.", g.Name, g.Gid)
		}
	}

}
