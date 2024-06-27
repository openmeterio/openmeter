package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/internal/entitlement"
	"github.com/openmeterio/openmeter/internal/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func getErrorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) bool {
		// user errors
		if _, ok := err.(*productcatalog.FeatureNotFoundError); ok {
			commonhttp.NewHTTPError(
				http.StatusNotFound,
				err,
			).EncodeError(ctx, w)
			return true
		}
		if _, ok := err.(*entitlement.NotFoundError); ok {
			commonhttp.NewHTTPError(
				http.StatusNotFound,
				err,
			).EncodeError(ctx, w)
			return true
		}
		if _, ok := err.(*models.GenericUserError); ok {
			commonhttp.NewHTTPError(
				http.StatusBadRequest,
				err,
			).EncodeError(ctx, w)
			return true
		}
		if err, ok := err.(*entitlement.AlreadyExistsError); ok {
			commonhttp.NewHTTPError(
				http.StatusConflict,
				err,
				commonhttp.ExtendProblem("conflictingEntityId", err.EntitlementID),
			).EncodeError(ctx, w)
			return true
		}
		if err, ok := err.(*entitlement.InvalidValueError); ok {
			commonhttp.NewHTTPError(
				http.StatusBadRequest,
				err,
			).EncodeError(ctx, w)
			return true
		}
		if err, ok := err.(*entitlement.InvalidFeatureError); ok {
			commonhttp.NewHTTPError(
				http.StatusBadRequest,
				err,
			).EncodeError(ctx, w)
			return true
		}
		// system errors (naming known errors for transparency)
		if _, ok := err.(*entitlement.WrongTypeError); ok {
			commonhttp.NewHTTPError(
				http.StatusInternalServerError,
				err,
			).EncodeError(ctx, w)
			return true
		}
		return false
	}
}
