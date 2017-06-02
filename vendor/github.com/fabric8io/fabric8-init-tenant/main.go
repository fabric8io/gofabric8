package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/Sirupsen/logrus"
	"github.com/almighty/almighty-core/log"
	"github.com/fabric8io/fabric8-init-tenant/app"
	"github.com/fabric8io/fabric8-init-tenant/configuration"
	"github.com/fabric8io/fabric8-init-tenant/controller"
	"github.com/fabric8io/fabric8-init-tenant/jsonapi"
	"github.com/fabric8io/fabric8-init-tenant/keycloak"
	"github.com/fabric8io/fabric8-init-tenant/migration"
	"github.com/fabric8io/fabric8-init-tenant/openshift"
	"github.com/fabric8io/fabric8-init-tenant/tenant"
	"github.com/goadesign/goa"
	"github.com/goadesign/goa/middleware"
	"github.com/goadesign/goa/middleware/gzip"
	goajwt "github.com/goadesign/goa/middleware/security/jwt"
	"github.com/jinzhu/gorm"

	goalogrus "github.com/goadesign/goa/logging/logrus"
	"github.com/spf13/viper"
)

func main() {

	viper.GetStringMapString("TEST")

	var migrateDB bool
	flag.BoolVar(&migrateDB, "migrateDatabase", false, "Migrates the database to the newest version and exits.")
	flag.Parse()

	// Initialized configuration
	config, err := configuration.GetData()
	if err != nil {
		logrus.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}

	// Initialized developer mode flag for the logger
	log.InitializeLogger(config.IsDeveloperModeEnabled())

	db := connect(config)
	defer db.Close()
	migrate(db)

	// Nothing to here except exit, since the migration is already performed.
	if migrateDB {
		os.Exit(0)
	}

	serviceToken := config.GetOpenshiftServiceToken()
	if serviceToken == "" {
		if config.UseOpenshiftCurrentCluster() {
			file, err := ioutil.ReadFile("/run/secrets/kubernetes.io/serviceaccount/token")
			if err != nil {
				logrus.Panic(nil, map[string]interface{}{
					"err": err,
				}, "failed to read service account token")
			}
			serviceToken = strings.TrimSpace(string(file))
		} else {
			logrus.Panic(nil, map[string]interface{}{}, "missing service token")
		}
	}

	var tr *http.Transport
	if config.APIServerInsecureSkipTLSVerify() {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	openshiftConfig := openshift.Config{
		MasterURL:     config.GetOpenshiftTenantMasterURL(),
		Token:         serviceToken,
		HttpTransport: tr,
	}

	openshiftMasterUser, err := openshift.WhoAmI(openshiftConfig)
	if err != nil {
		logrus.Panic(nil, map[string]interface{}{
			"err": err,
		}, "unknown master user based on service token")
	}
	openshiftConfig.MasterUser = openshiftMasterUser

	keycloakConfig := keycloak.Config{
		BaseURL: config.GetKeycloakURL(),
		Realm:   config.GetKeycloakRealm(),
		Broker:  config.GetKeycloakOpenshiftBroker(),
	}

	templateVars, err := config.GetTemplateValues()
	if err != nil {
		panic(err)
	}

	publicKey, err := keycloak.GetPublicKey(keycloakConfig)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to parse public token")
	}

	// Create service
	service := goa.New("tenant")

	// Mount middleware
	service.Use(middleware.RequestID())
	service.Use(middleware.LogRequest(config.IsDeveloperModeEnabled()))
	service.Use(gzip.Middleware(9))
	service.Use(jsonapi.ErrorHandler(service, true))
	service.Use(middleware.Recover())
	service.WithLogger(goalogrus.New(log.Logger()))
	app.UseJWTMiddleware(service, goajwt.New(publicKey, nil, app.NewJWTSecurity()))

	// Mount "status" controller
	statusCtrl := controller.NewStatusController(service, db)
	app.MountStatusController(service, statusCtrl)

	// Mount "tenant" controller
	tenantCtrl := controller.NewTenantController(service, tenant.NewDBService(db), keycloakConfig, openshiftConfig, templateVars)
	app.MountTenantController(service, tenantCtrl)

	log.Logger().Infoln("Git Commit SHA: ", controller.Commit)
	log.Logger().Infoln("UTC Build Time: ", controller.BuildTime)
	log.Logger().Infoln("UTC Start Time: ", controller.StartTime)
	log.Logger().Infoln("Dev mode:       ", config.IsDeveloperModeEnabled())

	http.Handle("/api/", service.Mux)
	http.Handle("/favicon.ico", http.NotFoundHandler())

	// Start http
	if err := http.ListenAndServe(config.GetHTTPAddress(), nil); err != nil {
		log.Error(nil, map[string]interface{}{
			"addr": config.GetHTTPAddress(),
			"err":  err,
		}, "unable to connect to server")
		service.LogError("startup", "err", err)
	}
}

func connect(config *configuration.Data) *gorm.DB {
	var err error
	var db *gorm.DB
	for {
		db, err = gorm.Open("postgres", config.GetPostgresConfigString())
		if err != nil {
			log.Logger().Errorf("ERROR: Unable to open connection to database %v", err)
			log.Logger().Infof("Retrying to connect in %v...", config.GetPostgresConnectionRetrySleep())
			time.Sleep(config.GetPostgresConnectionRetrySleep())
		} else {
			break
		}
	}

	if config.IsDeveloperModeEnabled() {
		db = db.Debug()
	}

	if config.GetPostgresConnectionMaxIdle() > 0 {
		log.Logger().Infof("Configured connection pool max idle %v", config.GetPostgresConnectionMaxIdle())
		db.DB().SetMaxIdleConns(config.GetPostgresConnectionMaxIdle())
	}
	if config.GetPostgresConnectionMaxOpen() > 0 {
		log.Logger().Infof("Configured connection pool max open %v", config.GetPostgresConnectionMaxOpen())
		db.DB().SetMaxOpenConns(config.GetPostgresConnectionMaxOpen())
	}
	return db
}

func migrate(db *gorm.DB) {
	// Migrate the schema
	err := migration.Migrate(db.DB())
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed migration")
	}
}
