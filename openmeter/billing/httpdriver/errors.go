package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[billingentity.NotFoundError](ctx, http.StatusNotFound, err, w, billingentity.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billingentity.ConflictError](ctx, http.StatusConflict, err, w, billingentity.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billingentity.ValidationError](ctx, http.StatusBadRequest, err, w, billingentity.EncodeValidationIssues) ||
			commonhttp.HandleErrorIfTypeMatches[billingentity.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w, billingentity.EncodeValidationIssues) ||
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
