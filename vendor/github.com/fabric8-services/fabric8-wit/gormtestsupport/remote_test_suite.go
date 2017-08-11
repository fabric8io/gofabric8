package gormtestsupport

import (
	config "github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"

	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &RemoteTestSuite{}
var _ suite.TearDownAllSuite = &RemoteTestSuite{}

// NewRemoteTestSuite instanciate a new RemoteTestSuite
func NewRemoteTestSuite(configFilePath string) RemoteTestSuite {
	return RemoteTestSuite{configFile: configFilePath}
}

// RemoteTestSuite is a base for tests using a gorm Remote
type RemoteTestSuite struct {
	suite.Suite
	configFile    string
	Configuration *config.ConfigurationData
}

// SetupSuite implements suite.SetupAllSuite
func (s *RemoteTestSuite) SetupSuite() {
	resource.Require(s.T(), resource.Remote)
	configuration, err := config.NewConfigurationData(s.configFile)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Configuration = configuration
}

// TearDownSuite implements suite.TearDownAllSuite
func (s *RemoteTestSuite) TearDownSuite() {
	s.Configuration = nil // Summon the GC!
}
