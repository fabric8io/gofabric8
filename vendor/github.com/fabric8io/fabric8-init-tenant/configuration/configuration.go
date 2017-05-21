package configuration

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	"encoding/base64"

	"github.com/spf13/viper"
)

const (
	// Constants for viper variable names. Will be used to set
	// default values as well as to get each value

	varPostgresHost                    = "postgres.host"
	varPostgresPort                    = "postgres.port"
	varPostgresUser                    = "postgres.user"
	varPostgresDatabase                = "postgres.database"
	varPostgresPassword                = "postgres.password"
	varPostgresSSLMode                 = "postgres.sslmode"
	varPostgresConnectionTimeout       = "postgres.connection.timeout"
	varPostgresConnectionRetrySleep    = "postgres.connection.retrysleep"
	varPostgresConnectionMaxIdle       = "postgres.connection.maxidle"
	varPostgresConnectionMaxOpen       = "postgres.connection.maxopen"
	varHTTPAddress                     = "http.address"
	varDeveloperModeEnabled            = "developer.mode.enabled"
	varKeycloakRealm                   = "keycloak.realm"
	varKeycloakOpenshiftBroker         = "keycloak.openshift.broker"
	varKeycloakURL                     = "keycloak.url"
	varOpenshiftTenantMasterURL        = "openshift.tenant.masterurl"
	varOpenshiftServiceToken           = "openshift.service.token"
	varTemplateRecommenderExternalName = "template.recommender.external.name"
	varTemplateRecommenderAPIToken     = "template.recommender.api.token"
	varTemplateDomain                  = "template.domain"
)

// Data encapsulates the Viper configuration object which stores the configuration data in-memory.
type Data struct {
	v *viper.Viper
}

// NewData creates a configuration reader object using a configurable configuration file path
func NewData() (*Data, error) {
	c := Data{
		v: viper.New(),
	}
	c.v.SetEnvPrefix("F8")
	c.v.AutomaticEnv()
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.v.SetTypeByDefaultValue(true)
	c.setConfigDefaults()

	return &c, nil
}

// String returns the current configuration as a string
func (c *Data) String() string {
	allSettings := c.v.AllSettings()
	y, err := yaml.Marshal(&allSettings)
	if err != nil {
		log.WithFields(map[string]interface{}{
			"settings": allSettings,
			"err":      err,
		}).Panicln("Failed to marshall config to string")
	}
	return fmt.Sprintf("%s\n", y)
}

// GetData is a wrapper over NewData which reads configuration file path
// from the environment variable.
func GetData() (*Data, error) {
	cd, err := NewData()
	return cd, err
}

func (c *Data) setConfigDefaults() {
	//---------
	// Postgres
	//---------
	c.v.SetTypeByDefaultValue(true)
	c.v.SetDefault(varPostgresHost, "localhost")
	c.v.SetDefault(varPostgresPort, 5432)
	c.v.SetDefault(varPostgresUser, "postgres")
	c.v.SetDefault(varPostgresDatabase, "tenant")
	c.v.SetDefault(varPostgresPassword, "mysecretpassword")
	c.v.SetDefault(varPostgresSSLMode, "disable")
	c.v.SetDefault(varPostgresConnectionTimeout, 5)
	c.v.SetDefault(varPostgresConnectionMaxIdle, -1)
	c.v.SetDefault(varPostgresConnectionMaxOpen, -1)

	// Number of seconds to wait before trying to connect again
	c.v.SetDefault(varPostgresConnectionRetrySleep, time.Duration(time.Second))

	//-----
	// HTTP
	//-----
	c.v.SetDefault(varHTTPAddress, "0.0.0.0:8080")

	//-----
	// Misc
	//-----
	c.v.SetDefault(varKeycloakOpenshiftBroker, defaultKeycloakOpenshiftBroker)

	// Enable development related features, e.g. token generation endpoint
	c.v.SetDefault(varDeveloperModeEnabled, false)

	// HTTP Cache-Control/max-age default
	c.v.SetDefault(varOpenshiftTenantMasterURL, defaultOpenshiftTenantMasterURL)
}

// GetPostgresHost returns the postgres host as set via default, config file, or environment variable
func (c *Data) GetPostgresHost() string {
	return c.v.GetString(varPostgresHost)
}

// GetPostgresPort returns the postgres port as set via default, config file, or environment variable
func (c *Data) GetPostgresPort() int64 {
	return c.v.GetInt64(varPostgresPort)
}

// GetPostgresUser returns the postgres user as set via default, config file, or environment variable
func (c *Data) GetPostgresUser() string {
	return c.v.GetString(varPostgresUser)
}

// GetPostgresDatabase returns the postgres database as set via default, config file, or environment variable
func (c *Data) GetPostgresDatabase() string {
	return c.v.GetString(varPostgresDatabase)
}

// GetPostgresPassword returns the postgres password as set via default, config file, or environment variable
func (c *Data) GetPostgresPassword() string {
	return c.v.GetString(varPostgresPassword)
}

// GetPostgresSSLMode returns the postgres sslmode as set via default, config file, or environment variable
func (c *Data) GetPostgresSSLMode() string {
	return c.v.GetString(varPostgresSSLMode)
}

// GetPostgresConnectionTimeout returns the postgres connection timeout as set via default, config file, or environment variable
func (c *Data) GetPostgresConnectionTimeout() int64 {
	return c.v.GetInt64(varPostgresConnectionTimeout)
}

// GetPostgresConnectionRetrySleep returns the number of seconds (as set via default, config file, or environment variable)
// to wait before trying to connect again
func (c *Data) GetPostgresConnectionRetrySleep() time.Duration {
	return c.v.GetDuration(varPostgresConnectionRetrySleep)
}

// GetPostgresConnectionMaxIdle returns the number of connections that should be keept alive in the database connection pool at
// any given time. -1 represents no restrictions/default behavior
func (c *Data) GetPostgresConnectionMaxIdle() int {
	return c.v.GetInt(varPostgresConnectionMaxIdle)
}

// GetPostgresConnectionMaxOpen returns the max number of open connections that should be open in the database connection pool.
// -1 represents no restrictions/default behavior
func (c *Data) GetPostgresConnectionMaxOpen() int {
	return c.v.GetInt(varPostgresConnectionMaxOpen)
}

// GetPostgresConfigString returns a ready to use string for usage in sql.Open()
func (c *Data) GetPostgresConfigString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		c.GetPostgresHost(),
		c.GetPostgresPort(),
		c.GetPostgresUser(),
		c.GetPostgresPassword(),
		c.GetPostgresDatabase(),
		c.GetPostgresSSLMode(),
		c.GetPostgresConnectionTimeout(),
	)
}

// GetHTTPAddress returns the HTTP address (as set via default, config file, or environment variable)
// that the alm server binds to (e.g. "0.0.0.0:8080")
func (c *Data) GetHTTPAddress() string {
	return c.v.GetString(varHTTPAddress)
}

// IsDeveloperModeEnabled returns if development related features (as set via default, config file, or environment variable),
// e.g. token generation endpoint are enabled
func (c *Data) IsDeveloperModeEnabled() bool {
	return c.v.GetBool(varDeveloperModeEnabled)
}

// GetKeycloakRealm returns the keyclaok realm name
func (c *Data) GetKeycloakRealm() string {
	if c.v.IsSet(varKeycloakRealm) {
		return c.v.GetString(varKeycloakRealm)
	}
	if c.IsDeveloperModeEnabled() {
		return devModeKeycloakRealm
	}
	return defaultKeycloakRealm
}

// GetKeycloakOpenshiftBroker returns the keyclaok broker name for openshift
func (c *Data) GetKeycloakOpenshiftBroker() string {
	return c.v.GetString(varKeycloakOpenshiftBroker)
}

// GetKeycloakURL returns Keycloak URL used by default in Dev mode
func (c *Data) GetKeycloakURL() string {
	if c.v.IsSet(varKeycloakURL) {
		return c.v.GetString(varKeycloakURL)
	}
	if c.IsDeveloperModeEnabled() {
		return devModeKeycloakURL
	}
	return defaultKeycloakURL
}

// GetOpenshiftTenantMasterURL returns the URL for the openshift cluster where the tenant services are running
func (c *Data) GetOpenshiftTenantMasterURL() string {
	return c.v.GetString(varOpenshiftTenantMasterURL)
}

// GetOpenshiftServiceToken returns the token be used by matser user for tenant init
func (c *Data) GetOpenshiftServiceToken() string {
	return c.v.GetString(varOpenshiftServiceToken)
}

// GetTemplateValues return a Map of additional variables used to process the templates
func (c *Data) GetTemplateValues() (map[string]string, error) {
	if !c.v.IsSet(varTemplateRecommenderExternalName) {
		return nil, fmt.Errorf("Missing required configuration %v", varTemplateRecommenderExternalName)
	}
	if !c.v.IsSet(varTemplateRecommenderAPIToken) {
		return nil, fmt.Errorf("Missing required configuration %v", varTemplateRecommenderAPIToken)
	}
	if !c.v.IsSet(varTemplateDomain) {
		return nil, fmt.Errorf("Missing required configuration %v", varTemplateDomain)
	}

	return map[string]string{
		"RECOMMENDER_EXTERNAL_NAME": c.v.GetString(varTemplateRecommenderExternalName),
		"RECOMMENDER_API_TOKEN":     base64.StdEncoding.EncodeToString([]byte(c.v.GetString(varTemplateRecommenderAPIToken))),
		"DOMAIN":                    c.v.GetString(varTemplateDomain),
	}, nil
}

const (
	// Auth-related defaults

	defaultKeycloakURL             = "https://sso.prod-preview.openshift.io"
	defaultKeycloakRealm           = "fabric8"
	defaultKeycloakOpenshiftBroker = "openshift-v3"

	// Keycloak vars to be used in dev mode. Can be overridden by setting up keycloak.url & keycloak.realm
	devModeKeycloakURL   = "https://sso.prod-preview.openshift.io"
	devModeKeycloakRealm = "fabric8-test"

	defaultOpenshiftTenantMasterURL = "https://api.free-int.openshift.com"
)
