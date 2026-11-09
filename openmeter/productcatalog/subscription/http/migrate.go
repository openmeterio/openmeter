package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	MigrateSubscriptionRequest  = plansubscription.MigrateSubscriptionRequest
	MigrateSubscriptionResponse = api.SubscriptionChangeResponseBody
	MigrateSubscriptionParams   = struct {
		ID string
	}
	MigrateSubscriptionHandler = httptransport.HandlerWithArgs[MigrateSubscriptionRequest, MigrateSubscriptionResponse, MigrateSubscriptionParams]
)

func (h *handler) MigrateSubscription() MigrateSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params MigrateSubscriptionParams) (MigrateSubscriptionRequest, error) {
			var body api.MigrateSubscriptionJSONRequestBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return MigrateSubscriptionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return MigrateSubscriptionRequest{}, err
			}

			var timing *subscription.Timing

			if body.Timing != nil {
				t, err := MapAPITimingToTiming(*body.Timing)
				if err != nil {
					return MigrateSubscriptionRequest{}, err
				}

				timing = &t
			}

			return MigrateSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				TargetVersion:    body.TargetVersion,
				StartingPhase:    body.StartingPhase,
				Timing:           timing,
				BillingAnchor:    body.BillingAnchor,
				RejectUnitConfig: true,
			}, nil
		},
		func(ctx context.Context, request MigrateSubscriptionRequest) (MigrateSubscriptionResponse, error) {
			var resp MigrateSubscriptionResponse

			// One transaction for the migration and the response lookup: if the lookup failed
			// after the migration committed, a retry would find the subscription already at
			// the target version and fail. Same as in change.go.
			err := h.TaxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				res, err := h.PlanSubscriptionService.Migrate(ctx, request)
				if err != nil {
					return err
				}

				taxCodes, err := h.taxCodesForView(ctx, res.Next)
				if err != nil {
					return err
				}

				v, err := MapSubscriptionViewToAPI(res.Next, taxCodes)
				if err != nil {
					return err
				}

				resp = MigrateSubscriptionResponse{
					Current: MapSubscriptionToAPI(res.Current),
					Next:    v,
				}

				return nil
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[MigrateSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("MigrateSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
