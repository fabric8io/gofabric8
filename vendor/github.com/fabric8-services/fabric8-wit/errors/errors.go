package errors

import (
	"context"
	"fmt"

	errs "github.com/pkg/errors"
)

const (
	stBadParameterErrorMsg         = "Bad value for parameter '%s': '%v'"
	stBadParameterErrorExpectedMsg = "Bad value for parameter '%s': '%v' (expected: '%v')"
	stNotFoundErrorMsg             = "%s with id '%s' not found"
)

// Constants that can be used to identify internal server errors
const (
	ErrInternalDatabase = "database_error"
)

type simpleError struct {
	message string
}

func (err simpleError) Error() string {
	return err.message
}

// NewInternalError returns the custom defined error of type InternalError.
func NewInternalError(ctx context.Context, err error) InternalError {
	return InternalError{err}
}

// IsInternalError returns true if the cause of the given error can be
// converted to an InternalError, which is returned as the second result.
func IsInternalError(err error) (bool, error) {
	e, ok := errs.Cause(err).(InternalError)
	if !ok {
		return false, nil
	}
	return true, e
}

// NewUnauthorizedError returns the custom defined error of type UnauthorizedError.
func NewUnauthorizedError(msg string) UnauthorizedError {
	return UnauthorizedError{simpleError{msg}}
}

// IsUnauthorizedError returns true if the cause of the given error can be
// converted to an UnauthorizedError, which is returned as the second result.
func IsUnauthorizedError(err error) (bool, error) {
	e, ok := errs.Cause(err).(UnauthorizedError)
	if !ok {
		return false, nil
	}
	return true, e
}

// NewForbiddenError returns the custom defined error of type ForbiddenError.
func NewForbiddenError(msg string) ForbiddenError {
	return ForbiddenError{simpleError{msg}}
}

// IsForbiddenError returns true if the cause of the given error can be
// converted to an ForbiddenError, which is returned as the second result.
func IsForbiddenError(err error) (bool, error) {
	e, ok := errs.Cause(err).(ForbiddenError)
	if !ok {
		return false, nil
	}
	return true, e
}

// InternalError means that the operation failed for some internal, unexpected reason
type InternalError struct {
	Err error
}

func (ie InternalError) Error() string {
	return ie.Err.Error()
}

// UnauthorizedError means that the operation is unauthorized
type UnauthorizedError struct {
	simpleError
}

// ForbiddenError means that the operation is forbidden
type ForbiddenError struct {
	simpleError
}

// VersionConflictError means that the version was not as expected in an update operation
type VersionConflictError struct {
	simpleError
}

// NewVersionConflictError returns the custom defined error of type VersionConflictError.
func NewVersionConflictError(msg string) VersionConflictError {
	return VersionConflictError{simpleError{msg}}
}

// DataConflictError means that the version was not as expected in an update operation
type DataConflictError struct {
	simpleError
}

// IsDataConflictError returns true if the cause of the given error can be
// converted to an IsDataConflictError, which is returned as the second result.
func IsDataConflictError(err error) (bool, error) {
	e, ok := errs.Cause(err).(DataConflictError)
	if !ok {
		return false, nil
	}
	return true, e
}

// NewDataConflictError returns the custom defined error of type NewDataConflictError.
func NewDataConflictError(msg string) DataConflictError {
	return DataConflictError{simpleError{msg}}
}

// IsVersionConflictError returns true if the cause of the given error can be
// converted to an VersionConflictError, which is returned as the second result.
func IsVersionConflictError(err error) (bool, error) {
	e, ok := errs.Cause(err).(VersionConflictError)
	if !ok {
		return false, nil
	}
	return true, e
}

// BadParameterError means that a parameter was not as required
type BadParameterError struct {
	parameter        string
	value            interface{}
	expectedValue    interface{}
	hasExpectedValue bool
}

// Error implements the error interface
func (err BadParameterError) Error() string {
	if err.hasExpectedValue {
		return fmt.Sprintf(stBadParameterErrorExpectedMsg, err.parameter, err.value, err.expectedValue)
	}
	return fmt.Sprintf(stBadParameterErrorMsg, err.parameter, err.value)

}

// Expected sets the optional expectedValue parameter on the BadParameterError
func (err BadParameterError) Expected(expexcted interface{}) BadParameterError {
	err.expectedValue = expexcted
	err.hasExpectedValue = true
	return err
}

// NewBadParameterError returns the custom defined error of type NewBadParameterError.
func NewBadParameterError(param string, actual interface{}) BadParameterError {
	return BadParameterError{parameter: param, value: actual}
}

// IsBadParameterError returns true if the cause of the given error can be
// converted to an BadParameterError, which is returned as the second result.
func IsBadParameterError(err error) (bool, error) {
	e, ok := errs.Cause(err).(BadParameterError)
	if !ok {
		return false, nil
	}
	return true, e
}

// NewConversionError returns the custom defined error of type NewConversionError.
func NewConversionError(msg string) ConversionError {
	return ConversionError{simpleError{msg}}
}

// IsConversionError returns true if the cause of the given error can be
// converted to an ConversionError, which is returned as the second result.
func IsConversionError(err error) (bool, error) {
	e, ok := errs.Cause(err).(ConversionError)
	if !ok {
		return false, nil
	}
	return true, e
}

// ConversionError error means something went wrong converting between different representations
type ConversionError struct {
	simpleError
}

// NotFoundError means the object specified for the operation does not exist
type NotFoundError struct {
	entity string
	ID     string
}

func (err NotFoundError) Error() string {
	return fmt.Sprintf(stNotFoundErrorMsg, err.entity, err.ID)
}

// NewNotFoundError returns the custom defined error of type NewNotFoundError.
func NewNotFoundError(entity string, id string) NotFoundError {
	return NotFoundError{entity: entity, ID: id}
}

// IsNotFoundError returns true if the cause of the given error can be
// converted to an NotFoundError, which is returned as the second result.
func IsNotFoundError(err error) (bool, error) {
	e, ok := errs.Cause(err).(NotFoundError)
	if !ok {
		return false, nil
	}
	return true, e
}
