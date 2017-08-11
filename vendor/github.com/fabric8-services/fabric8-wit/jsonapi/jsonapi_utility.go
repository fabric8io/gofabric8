package jsonapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

const (
	ErrorCodeNotFound          = "not_found"
	ErrorCodeBadParameter      = "bad_parameter"
	ErrorCodeVersionConflict   = "version_conflict"
	ErrorCodeUnknownError      = "unknown_error"
	ErrorCodeConversionError   = "conversion_error"
	ErrorCodeInternalError     = "internal_error"
	ErrorCodeUnauthorizedError = "unauthorized_error"
	ErrorCodeForbiddenError    = "forbidden_error"
	ErrorCodeJWTSecurityError  = "jwt_security_error"
	ErrorCodeDataConflict      = "data_conflict_error"
)

// ErrorToJSONAPIError returns the JSONAPI representation
// of an error and the HTTP status code that will be associated with it.
// This function knows about the models package and the errors from there
// as well as goa error classes.
func ErrorToJSONAPIError(ctx context.Context, err error) (app.JSONAPIError, int) {
	cause := errs.Cause(err)
	detail := cause.Error()
	var title, code string
	var statusCode int
	var id *string
	log.Error(ctx, map[string]interface{}{"err": cause, "error_message": cause.Error()}, "an error occurred in our api")
	switch cause.(type) {
	case errors.NotFoundError:
		code = ErrorCodeNotFound
		title = "Not found error"
		statusCode = http.StatusNotFound
	case errors.ConversionError:
		code = ErrorCodeConversionError
		title = "Conversion error"
		statusCode = http.StatusBadRequest
	case errors.BadParameterError:
		code = ErrorCodeBadParameter
		title = "Bad parameter error"
		statusCode = http.StatusBadRequest
	case errors.VersionConflictError:
		code = ErrorCodeVersionConflict
		title = "Version conflict error"
		statusCode = http.StatusConflict
	case errors.DataConflictError:
		code = ErrorCodeDataConflict
		title = "Data conflict error"
		statusCode = http.StatusConflict
	case errors.InternalError:
		code = ErrorCodeInternalError
		title = "Internal error"
		statusCode = http.StatusInternalServerError
	case errors.UnauthorizedError:
		code = ErrorCodeUnauthorizedError
		title = "Unauthorized error"
		statusCode = http.StatusUnauthorized
	case errors.ForbiddenError:
		code = ErrorCodeForbiddenError
		title = "Forbidden error"
		statusCode = http.StatusForbidden
	default:
		code = ErrorCodeUnknownError
		title = "Unknown error"
		statusCode = http.StatusInternalServerError

		cause := errs.Cause(err)
		if err, ok := cause.(goa.ServiceError); ok {
			statusCode = err.ResponseStatus()
			idStr := err.Token()
			id = &idStr
			title = http.StatusText(statusCode)
		}
		if errResp, ok := cause.(*goa.ErrorResponse); ok {
			code = errResp.Code
			detail = errResp.Detail
		}
	}
	statusCodeStr := strconv.Itoa(statusCode)
	jerr := app.JSONAPIError{
		ID:     id,
		Code:   &code,
		Status: &statusCodeStr,
		Title:  &title,
		Detail: detail,
	}
	return jerr, statusCode
}

// ErrorToJSONAPIErrors is a convenience function if you
// just want to return one error from the models package as a JSONAPI errors
// array.
func ErrorToJSONAPIErrors(ctx context.Context, err error) (*app.JSONAPIErrors, int) {
	jerr, httpStatusCode := ErrorToJSONAPIError(ctx, err)
	jerrors := app.JSONAPIErrors{}
	jerrors.Errors = append(jerrors.Errors, &jerr)
	return &jerrors, httpStatusCode
}

// BadRequest represent a Context that can return a BadRequest HTTP status
type BadRequest interface {
	BadRequest(*app.JSONAPIErrors) error
}

// InternalServerError represent a Context that can return a InternalServerError HTTP status
type InternalServerError interface {
	InternalServerError(*app.JSONAPIErrors) error
}

// NotFound represent a Context that can return a NotFound HTTP status
type NotFound interface {
	NotFound(*app.JSONAPIErrors) error
}

// Unauthorized represent a Context that can return a Unauthorized HTTP status
type Unauthorized interface {
	Unauthorized(*app.JSONAPIErrors) error
}

// Forbidden represent a Context that can return a Unauthorized HTTP status
type Forbidden interface {
	Forbidden(*app.JSONAPIErrors) error
}

// Conflict represent a Context that can return a Conflict HTTP status
type Conflict interface {
	Conflict(*app.JSONAPIErrors) error
}

// JSONErrorResponse auto maps the provided error to the correct response type
// If all else fails, InternalServerError is returned
func JSONErrorResponse(obj interface{}, err error) error {
	x := obj.(InternalServerError)
	c := obj.(context.Context)

	jsonErr, status := ErrorToJSONAPIErrors(c, err)
	switch status {
	case http.StatusBadRequest:
		if ctx, ok := x.(BadRequest); ok {
			return errs.WithStack(ctx.BadRequest(jsonErr))
		}
	case http.StatusNotFound:
		if ctx, ok := x.(NotFound); ok {
			return errs.WithStack(ctx.NotFound(jsonErr))
		}
	case http.StatusUnauthorized:
		if ctx, ok := x.(Unauthorized); ok {
			return errs.WithStack(ctx.Unauthorized(jsonErr))
		}
	case http.StatusForbidden:
		if ctx, ok := x.(Forbidden); ok {
			return errs.WithStack(ctx.Forbidden(jsonErr))
		}
	case http.StatusConflict:
		if ctx, ok := x.(Conflict); ok {
			return errs.WithStack(ctx.Conflict(jsonErr))
		}
	default:
		return errs.WithStack(x.InternalServerError(jsonErr))
	}
	return nil
}
