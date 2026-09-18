package customersentitlements

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

// errorEncoder maps entitlement domain errors to their HTTP responses
// and falls back to the generic models errors the customer service wraps (not
// found, precondition failed), which apierrors.GenericErrorEncoder does not handle.
func errorEncoder() encoder.ErrorEncoder {
	generic := commonhttp.GenericErrorEncoder()

	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		return commonhttp.HandleErrorIfTypeMatches[*entitlement.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.AlreadyExistsError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.InvalidValueError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.InvalidFeatureError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.WrongTypeError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.ForbiddenError](ctx, http.StatusForbidden, err, w) ||
			generic(ctx, err, w, r)
	}
}
