package subscriptions

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
)

// subscriptionGenericErrorEncoder maps the models.Generic* errors surfaced by the
// subscription command/workflow services to their HTTP statuses, then defers to
// the generic v3 encoder. These types are plain error wrappers, not
// ValidationIssues, so apierrors.GenericErrorEncoder does not classify them and
// ordinary client mistakes would otherwise fall through to a 500. Mirrors
// editSubscriptionErrorEncoder and the subscriptionaddons error encoder.
//
// Forbidden maps to 403: the subscription state machine returns a
// GenericForbiddenError when an action is not allowed in the current state (e.g.
// deleting an active/started subscription), which is a not-allowed action rather
// than a conflict.
func subscriptionGenericErrorEncoder() encoder.ErrorEncoder {
	generic := apierrors.GenericErrorEncoder()

	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericNotFoundError](ctx, http.StatusNotFound, err, w) ||
			generic(ctx, err, w, r)
	}
}
