package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/portal"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*portal.NotImplementedError](ctx, http.StatusNotImplemented, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*meter.MeterNotFoundError](ctx, http.StatusNotFound, err, w) ||
			// TODO: add generic errors to the common error handler
			commonhttp.HandleErrorIfTypeMatches[*models.GenericValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericNotImplementedError](ctx, http.StatusNotImplemented, err, w)
	}
}
