package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		// Generic errors
		return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w) ||
			// Subscription errors
			commonhttp.HandleErrorIfTypeMatches[*subscription.ForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.ItemNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PhaseNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PlanNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscriptionentitlement.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			// dependency: entitlement
			commonhttp.HandleErrorIfTypeMatches[*entitlement.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.AlreadyExistsError](
				ctx, http.StatusConflict, err, w,
				func(specificErr *entitlement.AlreadyExistsError) map[string]interface{} {
					return map[string]interface{}{
						"conflictingEntityId": specificErr.EntitlementID,
					}
				}) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.InvalidValueError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.InvalidFeatureError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*entitlement.WrongTypeError](ctx, http.StatusBadRequest, err, w) ||
			// dependency: plan
			commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusBadRequest, err, w)
	}
}
