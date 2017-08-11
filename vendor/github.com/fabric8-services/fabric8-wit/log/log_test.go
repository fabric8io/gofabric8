package log

import (
	"bytes"
	"encoding/json"
	"testing"

	logrus "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func LogAndAssertJSON(t *testing.T, log func(), assertions func(fields logrus.Fields)) {
	var buffer bytes.Buffer
	var fields logrus.Fields

	InitializeLogger(true, "debug")
	logger.Out = &buffer
	logger.Level = logrus.DebugLevel
	log()

	err := json.Unmarshal(buffer.Bytes(), &fields)
	assert.Nil(t, err)

	assertions(fields)
}

func TestInfo(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Info(nil, nil, "test")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "test")
		assert.Equal(t, fields["level"], "info")
		assert.Equal(t, fields["pkg"], "log.TestInfo")
	})
}

func TestInfoWithFields(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Info(nil, map[string]interface{}{"key": "value"}, "test")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "test")
		assert.Equal(t, fields["level"], "info")
		assert.Equal(t, fields["key"], "value")
		assert.Equal(t, fields["pkg"], "log.TestInfoWithFields")
	})
}

func TestWarn(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Warn(nil, nil, "test")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "test")
		assert.Equal(t, fields["level"], "warning")
	})
}

func TestDebug(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Debug(nil, nil, "test")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "test")
		assert.Equal(t, fields["level"], "debug")
	})
}

func TestDebugMsgFieldHasPrefix(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Debug(nil, map[string]interface{}{"req": "PUT", "info": "hello"}, "msg with additional fields: %s", "value of my field")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "msg with additional fields: value of my field")
		assert.Equal(t, fields["req"], "PUT")
		assert.Equal(t, fields["info"], "hello")
	})
}

func TestInfoMsgFieldHasPrefix(t *testing.T) {
	LogAndAssertJSON(t, func() {
		Info(nil, map[string]interface{}{"req": "GET"}, "message with additional fields: %s", "value of my field")
	}, func(fields logrus.Fields) {
		assert.Equal(t, fields["msg"], "message with additional fields: value of my field")
		assert.Equal(t, fields["req"], "GET")
		assert.Equal(t, fields["level"], "info")
	})
}
