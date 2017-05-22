package jsonapi_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/resource"
	"github.com/fabric8io/fabric8-init-tenant/app"
	"github.com/fabric8io/fabric8-init-tenant/jsonapi"
	"github.com/stretchr/testify/require"
)

func TestErrorToJSONAPIError(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)

	var jerr app.JSONAPIError
	var httpStatus int

	// test not found error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(errors.NewNotFoundError("foo", "bar"))
	require.Equal(t, http.StatusNotFound, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeNotFound, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)

	// test not found error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(errors.NewConversionError("foo"))
	require.Equal(t, http.StatusBadRequest, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeConversionError, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)

	// test bad parameter error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(errors.NewBadParameterError("foo", "bar"))
	require.Equal(t, http.StatusBadRequest, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeBadParameter, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)

	// test internal server error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(errors.NewInternalError("foo"))
	require.Equal(t, http.StatusInternalServerError, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeInternalError, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)

	// test unauthorized error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(errors.NewUnauthorizedError("foo"))
	require.Equal(t, http.StatusUnauthorized, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeUnauthorizedError, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)

	// test unspecified error
	jerr, httpStatus = jsonapi.ErrorToJSONAPIError(fmt.Errorf("foobar"))
	require.Equal(t, http.StatusInternalServerError, httpStatus)
	require.NotNil(t, jerr.Code)
	require.NotNil(t, jerr.Status)
	require.Equal(t, jsonapi.ErrorCodeUnknownError, *jerr.Code)
	require.Equal(t, strconv.Itoa(httpStatus), *jerr.Status)
}
