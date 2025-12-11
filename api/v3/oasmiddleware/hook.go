package oasmiddleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

var ErrRouteNotFound = errors.New("route not found")

// OasRouteNotFoundErrorHook handles the error when a route is not found in a validation
// router. This will stop the request lifecycle and return an AIP compliant 404 response
func OasRouteNotFoundErrorHook(err error, w http.ResponseWriter, r *http.Request) bool {
	if err != nil {
		apierrors.
			NewNotFoundError(r.Context(), ErrRouteNotFound, "route").
			HandleAPIError(w, r)
		return true
	}
	return false
}

// OasValidationErrorHook handles the error when a request is not matching the
// OAS spec definition for a given route in the validation router.
// This will stop the request lifecycle and return an AIP compliant 400 response
func OasValidationErrorHook(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
	switch err := err.(type) {
	case nil:
		return false
	case openapi3.MultiError:
		invalidParams := ToAipError(err)
		sourcePath := false
		for _, v := range invalidParams {
			if v.Source == apierrors.InvalidParamSourcePath {
				sourcePath = true
				break
			}
		}
		if sourcePath {
			apierrors.
				NewNotFoundError(ctx, err, "entity").
				HandleAPIError(w, r)
		} else {
			apierrors.
				NewBadRequestError(ctx, SanitizeSensitiveFieldValues(err), invalidParams).
				HandleAPIError(w, r)
		}
		return true
	case *openapi3filter.RequestError:
		if err.Parameter != nil && err.Parameter.In == "path" {
			apierrors.
				NewNotFoundError(ctx, err, "entity").
				HandleAPIError(w, r)
			return true
		}
	}
	apierrors.
		NewBadRequestError(ctx, err, nil).
		HandleAPIError(w, r)
	return true
}

func SanitizeSensitiveFieldValues(err error) error {
	switch err := err.(type) {
	case nil:
		return nil
	case openapi3.MultiError:
		sanitizedMultiErr := make(openapi3.MultiError, 0)
		for _, vErr := range err {
			sanitizedMultiErr = append(sanitizedMultiErr, SanitizeSensitiveFieldValues(vErr))
		}
		return sanitizedMultiErr
	case *openapi3filter.RequestError:
		err.Err = SanitizeSensitiveFieldValues(err.Err)
		return err
	case *openapi3.SchemaError:
		if err.Schema != nil && err.Schema.Extensions != nil {
			xSensitive, ok := err.Schema.Extensions["x-sensitive"]
			if ok && isSensitive(xSensitive) {
				err.Value = "********"
			}
		}
		return err
	default:
		return err
	}
}

func isSensitive(sensitive any) bool {
	switch v := sensitive.(type) {
	case string:
		if v == "true" {
			return true
		}
		return false
	case bool:
		return v
	default:
		return false
	}
}
