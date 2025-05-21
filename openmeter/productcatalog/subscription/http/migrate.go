package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
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

			timing, err := MapAPITimingToTiming(*body.Timing)
			if err != nil {
				return MigrateSubscriptionRequest{}, err
			}

			return MigrateSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				TargetVersion: body.TargetVersion,
				StartingPhase: body.StartingPhase,
				Timing:        timing,
			}, nil
		},
		func(ctx context.Context, request MigrateSubscriptionRequest) (MigrateSubscriptionResponse, error) {
			res, err := h.PlanSubscriptionService.Migrate(ctx, request)
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			v, err := MapSubscriptionViewToAPI(res.Next)

			return MigrateSubscriptionResponse{
				Current: MapSubscriptionToAPI(res.Current),
				Next:    v,
			}, err
		},
		commonhttp.JSONResponseEncoderWithStatus[MigrateSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("MigrateSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
