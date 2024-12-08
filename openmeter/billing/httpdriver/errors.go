package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ConflictError](ctx, http.StatusConflict, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationError](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w, billing.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billing.ValidationIssue](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues) ||
			// dependency: customer
			commonhttp.HandleErrorIfTypeMatches[customerentity.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.ValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.SubjectKeyConflictError](ctx, http.StatusConflict, err, w) ||
			// dependency: apps
			commonhttp.HandleErrorIfTypeMatches[app.AppNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppDefaultNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.ValidationError](ctx, http.StatusBadRequest, err, w)
	}
}
