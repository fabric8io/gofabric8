package test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
)

// TestDataProvider defines the simple funcion for returning data from a remote provider
type TestDataProvider func() ([]byte, error)

// LoadTestData attempt to load test data from local disk unless;
// * It does not exist or,
// * Variable REFRESH_DATA is present in ENV
//
// Data is stored under examples/test
// This is done to avoid always depending on remote systems, but also with an option
// to refresh/retest against the 'current' remote system data without manual copy/paste
func LoadTestData(filename string, provider TestDataProvider) ([]byte, error) {
	refreshLocalData := func(path string, refresh TestDataProvider) ([]byte, error) {
		content, err := refresh()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		err = ioutil.WriteFile(path, content, 0644)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return content, nil
	}

	// Get path to src/github.com/fabric8-services/fabric8-wit/test/remote_test_data.go
	_, packagefilename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}

	// The target dir would be src/github.com/fabric8-services/fabric8-wit/examples/test
	targetDir := filepath.FromSlash(path.Dir(packagefilename) + "/../test/data/")
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	targetPath := filepath.FromSlash(targetDir + filename)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		// Call refresher if data does not exist locally
		return refreshLocalData(targetPath, provider)
	}
	if _, found := os.LookupEnv("REFRESH_DATA"); found {
		// Call refresher if force update of test data set in env
		return refreshLocalData(targetPath, provider)
	}

	return ioutil.ReadFile(targetPath)
}
