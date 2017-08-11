package errors

import (
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/stretchr/testify/assert"
)

func TestSimpleError_Error(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	e := simpleError{message: "foo"}
	assert.Equal(t, "foo", e.Error())
}

func TestBadParameterError_Error(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	e := BadParameterError{parameter: "foo", value: "bar"}
	assert.Equal(t, fmt.Sprintf(stBadParameterErrorMsg, e.parameter, e.value), e.Error())

	e = BadParameterError{parameter: "foo", value: "bar", expectedValue: "foobar", hasExpectedValue: true}
	assert.Equal(t, fmt.Sprintf(stBadParameterErrorExpectedMsg, e.parameter, e.value, e.expectedValue), e.Error())
}

func TestNotFoundError_Error(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	e := NotFoundError{entity: "foo", ID: "bar"}
	assert.Equal(t, fmt.Sprintf(stNotFoundErrorMsg, e.entity, e.ID), e.Error())
}
