package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		// TODO (pmarton): We need to find a better way to handle cross package errors instead of registering all of them here
		return commonhttp.HandleErrorIfTypeMatches[app.AppNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppDefaultNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderError](ctx, http.StatusFailedDependency, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderAuthenticationError](ctx, http.StatusUnauthorized, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderPreConditionError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppCustomerPreConditionError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.ValidationError](ctx, http.StatusBadRequest, err, w) ||

			// TODO (pmarton): We need to add this customer error because Stripe Checkout Session creation can create a customer and fail with this error
			// On the long term all errors should implement a common interface that controls the HTTP status code.
			commonhttp.HandleErrorIfTypeMatches[customerentity.SubjectKeyConflictError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w) ||

			// App errors
			commonhttp.HandleErrorIfTypeMatches[app.AppDefaultNotFoundError](ctx, http.StatusPreconditionFailed, err, w)
	}
}
