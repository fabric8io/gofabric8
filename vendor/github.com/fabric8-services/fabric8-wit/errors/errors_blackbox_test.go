package errors_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/resource"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInternalError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := errors.NewInternalError(context.Background(), errs.New("system disk could not be read"))

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewConversionError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	err := errors.NewConversionError("Couldn't convert workitem")

	// not sure what assertion to do here.
	t.Log(err)
}

func TestNewBadParameterError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := 10
	expectedValue := 11
	err := errors.NewBadParameterError(param, value)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v'", param, value), err.Error())
	err = errors.NewBadParameterError(param, value).Expected(expectedValue)
	assert.Equal(t, fmt.Sprintf("Bad value for parameter '%s': '%v' (expected: '%v')", param, value, expectedValue), err.Error())
}

func TestNewNotFoundError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	param := "assigness"
	value := "10"
	err := errors.NewNotFoundError(param, value)
	assert.Equal(t, fmt.Sprintf("%s with id '%s' not found", param, value), err.Error())
}

func TestNewUnauthorizedError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	msg := "Invalid token"
	err := errors.NewUnauthorizedError(msg)

	assert.Equal(t, msg, err.Error())
}

func TestNewForbiddenError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	msg := "Forbidden"
	err := errors.NewForbiddenError(msg)

	assert.Equal(t, msg, err.Error())
}

func TestIsXYError(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	t.Parallel()
	ctx := context.Background()
	testCases := []struct {
		name           string
		arg            error
		fn             func(err error) (bool, error)
		expectedResult bool
	}{
		{"IsInternalError - is an InternalError", errors.NewInternalError(ctx, errs.New("some message")), errors.IsInternalError, true},
		{"IsInternalError - is a wrapped InternalError", errs.Wrap(errs.Wrap(errors.NewInternalError(ctx, errs.New("some message")), "msg1"), "msg2"), errors.IsInternalError, true},
		{"IsInternalError - is not an InternalError", errors.NewNotFoundError("foo", "bar"), errors.IsInternalError, false},
		{"IsBadParameterError - is a BadParameterError", errors.NewBadParameterError("param", "actual"), errors.IsBadParameterError, true},
		{"IsBadParameterError - is a wrapped BadParameterError", errs.Wrap(errs.Wrap(errors.NewBadParameterError("param", "actual"), "msg1"), "msg2"), errors.IsBadParameterError, true},
		{"IsBadParameterError - is not a BadParameterError", errors.NewNotFoundError("foo", "bar"), errors.IsBadParameterError, false},
		{"IsConversionError - is a ConversionError", errors.NewConversionError("some message"), errors.IsConversionError, true},
		{"IsConversionError - is a wrapped ConversionError", errs.Wrap(errs.Wrap(errors.NewConversionError("some message"), "msg1"), "msg2"), errors.IsConversionError, true},
		{"IsConversionError - is not a ConversionError", errors.NewNotFoundError("foo", "bar"), errors.IsConversionError, false},
		{"IsForbiddenError - is a ForbiddenError", errors.NewForbiddenError("some message"), errors.IsForbiddenError, true},
		{"IsForbiddenError - is a wrapped ForbiddenError", errs.Wrap(errs.Wrap(errors.NewForbiddenError("some message"), "msg1"), "msg2"), errors.IsForbiddenError, true},
		{"IsForbiddenError - is not a ForbiddenError", errors.NewNotFoundError("foo", "bar"), errors.IsForbiddenError, false},
		{"IsNotFoundError - is a NotFoundError", errors.NewNotFoundError("entity", "id"), errors.IsNotFoundError, true},
		{"IsNotFoundError - is a wrapped NotFoundError", errs.Wrap(errs.Wrap(errors.NewNotFoundError("entity", "id"), "msg1"), "msg2"), errors.IsNotFoundError, true},
		{"IsNotFoundError - is not a NotFoundError", errors.NewInternalError(ctx, errs.New("some message")), errors.IsNotFoundError, false},
		{"IsUnauthorizedError - is an UnauthorizedError", errors.NewUnauthorizedError("some message"), errors.IsUnauthorizedError, true},
		{"IsUnauthorizedError - is a wrapped UnauthorizedError", errs.Wrap(errs.Wrap(errors.NewUnauthorizedError("some message"), "msg1"), "msg2"), errors.IsUnauthorizedError, true},
		{"IsUnauthorizedError - is not an UnauthorizedError", errors.NewInternalError(ctx, errs.New("some message")), errors.IsUnauthorizedError, false},
		{"IsVersionConflictError - is a VersionConflictError", errors.NewVersionConflictError("some message"), errors.IsVersionConflictError, true},
		{"IsVersionConflictError - is a wrapped VersionConflictError", errs.Wrap(errs.Wrap(errors.NewVersionConflictError("some message"), "msg1"), "msg2"), errors.IsVersionConflictError, true},
		{"IsVersionConflictError - is not a VersionConflictError", errors.NewInternalError(ctx, errs.New("some message")), errors.IsVersionConflictError, false},
	}
	for _, tc := range testCases {
		// Note that we need to capture the range variable to ensure that tc
		// gets bound to the correct instance.
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actualResult, err := tc.fn(tc.arg)
			require.Equal(t, tc.expectedResult, actualResult)
			require.Equal(t, tc.expectedResult, (err != nil))
		})
	}
}
