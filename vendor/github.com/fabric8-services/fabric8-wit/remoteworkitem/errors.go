package remoteworkitem

import "fmt"

type simpleError struct {
	message string
}

func (err simpleError) Error() string {
	return err.message
}

// InternalError means that the operation failed for some internal, unexpected reason
type InternalError struct {
	simpleError
}

// VersionConflictError means that the version was not as expected in an update operation
type VersionConflictError struct {
	simpleError
}

// BadParameterError means that a parameter was not as required
type BadParameterError struct {
	parameter string
	value     interface{}
}

// Error implements the error interface
func (err BadParameterError) Error() string {
	return fmt.Sprintf("Bad value for parameter '%s': '%v'", err.parameter, err.value)
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
	return fmt.Sprintf("%s with id '%s' not found", err.entity, err.ID)
}
