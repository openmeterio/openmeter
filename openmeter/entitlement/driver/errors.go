package entitlementdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func GetErrorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		// user errors
		return commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w) ||
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
			commonhttp.HandleErrorIfTypeMatches[*entitlement.WrongTypeError](ctx, http.StatusBadRequest, err, w)
	}
}
