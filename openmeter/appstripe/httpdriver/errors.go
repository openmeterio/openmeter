package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[appstripe.AppNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[appstripe.WebhookAppNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[appstripe.ValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[appstripe.StripeCustomerPreConditionError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w)
	}
}
