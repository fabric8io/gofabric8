package controller_test

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"path/filepath"

	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/resource"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var updateGoldenFiles = flag.Bool("update", false, "when set, rewrite the golden files")

// compareWithGolden compares the actual object against the one from a golden
// file. The comparison is done by marshalling the output to JSON and comparing
// on string level If the comparison fails, the given test will fail. If the
// -update flag is given, that golden file is overwritten with the current
// actual object. When adding new tests you first must run them with the -update
// flag in order to create an initial golden version.
func compareWithGolden(t *testing.T, goldenFile string, actualObj interface{}) {
	err := testableCompareWithGolden(*updateGoldenFiles, goldenFile, actualObj)
	if err != nil {
		t.Fatalf("%+v", err)
	}
}

func testableCompareWithGolden(update bool, goldenFile string, actualObj interface{}) error {
	absPath, err := filepath.Abs(goldenFile)
	if err != nil {
		return errs.WithStack(err)
	}
	actual, err := json.MarshalIndent(actualObj, "", "  ")
	if err != nil {
		return errs.WithStack(err)
	}
	if update {
		err = ioutil.WriteFile(absPath, actual, os.ModePerm)
		if err != nil {
			return errs.Wrapf(err, "failed to update golden file: %s", absPath)
		}
	}
	expected, err := ioutil.ReadFile(absPath)
	if err != nil {
		return errs.Wrapf(err, "failed to read golden file: %s", absPath)
	}

	expectedStr := string(expected)
	actualStr := string(actual)
	if expectedStr != actualStr {
		log.Error(nil, nil, "testableCompareWithGolden: expected value %v", expectedStr)
		log.Error(nil, nil, "testableCompareWithGolden: actual value %v", actualStr)

		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(expectedStr, actualStr, false)
		log.Error(nil, nil, "testableCompareWithGolden: mismatch of actual output and golden-file %s:\n %s \n", absPath, dmp.DiffPrettyText(diffs))
		return errs.Errorf("mismatch of actual output and golden-file %s:\n %s \n", absPath, dmp.DiffPrettyText(diffs))
	}
	return nil
}

func TestCompareWithGolden(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	type Foo struct {
		Bar string
	}
	dummy := Foo{Bar: "hello world"}
	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		// given
		f := "not_existing_file.golden.json"
		// when
		err := testableCompareWithGolden(false, f, dummy)
		// then
		require.NotNil(t, err)
		_, isPathError := errs.Cause(err).(*os.PathError)
		require.True(t, isPathError)
	})
	t.Run("unable to update golden file due to not existing folder", func(t *testing.T) {
		t.Parallel()
		// given
		f := "not/existing/folder/file.golden.json"
		// when
		err := testableCompareWithGolden(true, f, dummy)
		// then
		require.NotNil(t, err)
		_, isPathError := errs.Cause(err).(*os.PathError)
		require.True(t, isPathError)
	})
	t.Run("mismatch between expected and actual output", func(t *testing.T) {
		t.Parallel()
		// given
		f := "test-files/codebase/show/ok_without_auth.golden.json"
		// when
		err := testableCompareWithGolden(false, f, dummy)
		// then
		require.NotNil(t, err)
		_, isPathError := errs.Cause(err).(*os.PathError)
		require.False(t, isPathError)
	})
	t.Run("ok - expected output equals actual", func(t *testing.T) {
		t.Parallel()
		// given
		f := "test-files/dummy.golden.json"
		bs, err := json.MarshalIndent(dummy, "", "  ")
		require.Nil(t, err)
		err = ioutil.WriteFile(f, bs, os.ModePerm)
		require.Nil(t, err)
		defer func() {
			err := os.Remove(f)
			require.Nil(t, err)
		}()
		// when
		err = testableCompareWithGolden(false, f, dummy)
		// then
		require.Nil(t, err)
	})

}
