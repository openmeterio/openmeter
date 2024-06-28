package httpdriver

import (
	"context"
	"net/http"

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

		if _, ok := err.(*productcatalog.FeatureInvalidFiltersError); ok {
			commonhttp.NewHTTPError(
				http.StatusBadRequest,
				err,
			).EncodeError(ctx, w)
			return true
		}

		if _, ok := err.(*productcatalog.FeatureInvalidMeterAggregationError); ok {
			commonhttp.NewHTTPError(
				http.StatusBadRequest,
				err,
			).EncodeError(ctx, w)
			return true
		}

		if _, ok := err.(*models.MeterNotFoundError); ok {
			commonhttp.NewHTTPError(
				http.StatusNotFound,
				err,
			).EncodeError(ctx, w)
			return true
		}

		if _, ok := err.(*productcatalog.FeatureWithNameAlreadyExistsError); ok {
			commonhttp.NewHTTPError(
				http.StatusConflict,
				err,
			).EncodeError(ctx, w)
			return true
		}

		return false
	}
}
