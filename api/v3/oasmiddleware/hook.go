package oasmiddleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
)

var ErrRouteNotFound = errors.New("route not found")

// OasRouteNotFoundErrorHook handles the error when a route is not found in validation.
// It stops the request lifecycle and returns an AIP-compliant 404 response.
func OasRouteNotFoundErrorHook(err error, w http.ResponseWriter, r *http.Request) bool {
	if err != nil {
		apierrors.
			NewNotFoundError(r.Context(), ErrRouteNotFound, "route").
			HandleAPIError(w, r)
		return true
	}
	return false
}

// OasValidationErrorHook handles the error when a request does not match the OAS spec.
// It stops the request lifecycle and returns an AIP-compliant 400 or 404 response.
func OasValidationErrorHook(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
	if err == nil {
		return false
	}

	var ve *LibopenapiValidationErrors
	if !errors.As(err, &ve) {
		apierrors.
			NewBadRequestError(ctx, err, nil).
			HandleAPIError(w, r)
		return true
	}

	invalidParams := ToAipErrorFromLibopenapi(ve.Errors)
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
			NewBadRequestError(ctx, err, invalidParams).
			HandleAPIError(w, r)
	}
	return true
}
