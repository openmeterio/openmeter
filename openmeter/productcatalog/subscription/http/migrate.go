package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	MigrateSubscriptionRequest = struct {
		id            models.NamespacedID
		targetVersion int
	}
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

			return MigrateSubscriptionRequest{
				id: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				targetVersion: body.TargetVersion,
			}, nil
		},
		func(ctx context.Context, request MigrateSubscriptionRequest) (MigrateSubscriptionResponse, error) {
			// Let's fetch the current sub
			sub, err := h.SubscriptionService.Get(ctx, request.id)
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			if sub.PlanRef == nil {
				return MigrateSubscriptionResponse{}, &models.GenericUserError{
					Message: fmt.Sprintf("Subscription %s has no plan, cannot be migrated", request.id.ID),
				}
			}

			// Let's fetch the latest version of the
			plan, err := h.SubscrpiptionPlanAdapter.GetVersion(ctx, request.id.Namespace, plansubscription.PlanRefInput{
				Key:     sub.PlanRef.Key,
				Version: &request.targetVersion,
			})
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			// Then let's create the subscription from the plan
			curr, new, err := h.SubscriptionWorkflowService.ChangeToPlan(ctx, request.id, subscription.ChangeSubscriptionWorkflowInput{
				ActiveFrom:     clock.Now(),
				AnnotatedModel: sub.AnnotatedModel,
				Name:           sub.Name,
				Description:    sub.Description,
			}, plan)
			if err != nil {
				return MigrateSubscriptionResponse{}, err
			}

			v, err := MapSubscriptionViewToAPI(new)

			return MigrateSubscriptionResponse{
				Current: MapSubscriptionToAPI(curr),
				New:     v,
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
