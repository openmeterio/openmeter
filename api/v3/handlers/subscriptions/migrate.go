package subscriptions

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	MigrateSubscriptionRequest  = plansubscription.MigrateSubscriptionRequest
	MigrateSubscriptionResponse = api.BillingSubscriptionMigrateResponse
	MigrateSubscriptionParams   = string
	MigrateSubscriptionHandler  httptransport.HandlerWithArgs[MigrateSubscriptionRequest, MigrateSubscriptionResponse, MigrateSubscriptionParams]
)

func (h *handler) MigrateSubscription() MigrateSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID MigrateSubscriptionParams) (MigrateSubscriptionRequest, error) {
			var body api.BillingSubscriptionMigrate
			if err := request.ParseBody(r, &body); err != nil {
				return MigrateSubscriptionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return MigrateSubscriptionRequest{}, err
			}

			return FromAPIBillingSubscriptionMigrate(models.NamespacedID{Namespace: ns, ID: subscriptionID}, body)
		},
		func(ctx context.Context, req MigrateSubscriptionRequest) (MigrateSubscriptionResponse, error) {
			result, err := h.planSubscriptionService.Migrate(ctx, req)
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			next, err := ToAPIBillingSubscription(result.Next)
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			// Re-reading current would lose the before snapshot for in-place migrations.
			return MigrateSubscriptionResponse{
				Current: ToAPIBillingSubscriptionBase(result.Current),
				Next:    next,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MigrateSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("migrate-subscription"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
