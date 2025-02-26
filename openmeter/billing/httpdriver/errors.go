package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ConflictError](ctx, http.StatusConflict, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationError](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationIssue](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			// dependency: customer
			commonhttp.HandleErrorIfTypeMatches[customer.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customer.SubjectKeyConflictError](ctx, http.StatusConflict, err, w) ||
			// dependency: apps
			commonhttp.HandleErrorIfTypeMatches[billing.AppError](ctx, http.StatusBadRequest, err, w)
	}
}
