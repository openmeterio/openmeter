package subscriptionaddons

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

// subscriptionAddonErrorEncoder maps the errors surfaced by the subscription
// workflow service (AddAddon / ChangeAddonQuantity) to their HTTP statuses.
// Those methods return models.Generic* errors — validation (e.g. exceeding a
// single-instance or configured maximum quantity), conflict (duplicate add-on),
// and not-found (unknown add-on) — which the generic v3 encoder does not
// classify, so without this ordinary client mistakes would fall through to a
// 500. It maps those types first, then defers to the generic v3 encoder so
// v3-native errors keep their envelope. Mirrors editSubscriptionErrorEncoder in
// the parent package.
func subscriptionAddonErrorEncoder() encoder.ErrorEncoder {
	generic := apierrors.GenericErrorEncoder()

	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericNotFoundError](ctx, http.StatusNotFound, err, w) ||
			generic(ctx, err, w, r)
	}
}
