package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func errorEncoder() encoder.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
		issues, err := models.AsValidationIssues(err)
		if err == nil && len(issues) > 0 {
			// Let's map the FieldSelectors to the public schema
			mappedIssues, err := slicesx.MapWithErr(issues, func(issue models.ValidationIssue) (models.ValidationIssue, error) {
				return subscription.MapSubscriptionSpecValidationIssueField(issue)
			})
			if err != nil {
				return false // Server dies if mapping fails
			}

			// This should be cleaned up by implementing attributes to non-validation issues
			if errors.Is(err, subscription.ErrOnlySingleSubscriptionAllowed) {
				problem := models.NewStatusProblem(ctx, errors.New("conflict"), http.StatusConflict)
				problem.Extensions = map[string]interface{}{
					"conflicts": lo.Map(mappedIssues, func(issue models.ValidationIssue, _ int) map[string]interface{} {
						return issue.AsErrorExtension()
					}),
				}

				problem.Respond(w)
				return true
			}

			// And let's respond with an error
			problem := models.NewStatusProblem(ctx, errors.New("validation error"), http.StatusBadRequest)
			problem.Extensions = map[string]interface{}{
				"validationErrors": lo.Map(mappedIssues, func(issue models.ValidationIssue, _ int) map[string]interface{} {
					return issue.AsErrorExtension()
				}),
			}

			problem.Respond(w)
			return true
		}

		// Generic errors
		return commonhttp.HandleErrorIfTypeMatches[*pagination.InvalidError](ctx, http.StatusBadRequest, err, w) ||
			// Subscription errors
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchConflictError](ctx, http.StatusConflict, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchForbiddenError](ctx, http.StatusForbidden, err, w) ||
			commonhttp.HandleErrorIfTypeMatches[*subscription.PatchValidationError](ctx, http.StatusBadRequest, err, w) ||
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
