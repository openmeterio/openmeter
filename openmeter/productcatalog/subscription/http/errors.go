package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func errorEncoder() httptransport.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		// Generic errors
		return commonhttp.HandleErrorIfTypeMatches[*models.GenericUserError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*models.GenericConflictError](ctx, http.StatusConflict, err, w) ||
			// Subscription errors
			commonhttp.HandleErrorIfTypeMatches[*subscription.ForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.ItemNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PhaseNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PlanNotFoundError](ctx, http.StatusNotFound, err, w) ||
			// FIXME: dependency errors should not have to be matched everywhere
			// dependency: plan
			commonhttp.HandleErrorIfTypeMatches[*feature.FeatureNotFoundError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[plan.ValidationError](ctx, http.StatusBadRequest, err, w) ||
			// dependency: customer
			commonhttp.HandleErrorIfTypeMatches[customerentity.NotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.ValidationError](ctx, http.StatusBadRequest, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.UpdateAfterDeleteError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[customerentity.SubjectKeyConflictError](ctx, http.StatusConflict, err, w) ||
			// dependency: app (due to validators)
			commonhttp.HandleErrorIfTypeMatches[app.AppNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppDefaultNotFoundError](ctx, http.StatusNotFound, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderAuthenticationError](ctx, http.StatusUnauthorized, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderError](ctx, http.StatusFailedDependency, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppProviderPreConditionError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.AppCustomerPreConditionError](ctx, http.StatusPreconditionFailed, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[app.ValidationError](ctx, http.StatusBadRequest, err, w)
	}
}
