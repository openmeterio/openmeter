package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// TODO: might need or not need a single interface for using the multiple workflow methods
	CreateSubscriptionRequest = struct {
		inp  subscription.CreateSubscriptionWorkflowInput
		plan subscription.PlanRefInput
	}
	CreateSubscriptionResponse = api.Subscription
	// CreateSubscriptionParams   = api.CreateSubscriptionParams
	// CreateSubscriptionHandler  httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]
	CreateSubscriptionHandler = httptransport.Handler[CreateSubscriptionRequest, CreateSubscriptionResponse]
)

func (h *handler) CreateSubscription() CreateSubscriptionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateSubscriptionRequest, error) {
			body := api.CreateSubscriptionJSONRequestBody{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateSubscriptionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			planSubBody, errAsPlanSub := body.AsPlanSubscriptionCreate()
			_, errAsCustSub := body.AsCustomSubscriptionCreate()

			// Custom subscription creation is not currently supported
			if errAsPlanSub != nil && errAsCustSub == nil {
				return CreateSubscriptionRequest{}, commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("custom subscription creation is not supported"))
			}

			if errAsPlanSub != nil {
				return CreateSubscriptionRequest{}, errAsPlanSub
			}

			return CreateSubscriptionRequest{
				inp: subscription.CreateSubscriptionWorkflowInput{
					Namespace:   ns,
					ActiveFrom:  planSubBody.ActiveFrom,
					CustomerID:  planSubBody.CustomerId,
					Name:        planSubBody.Name,
					Description: planSubBody.Description,
					AnnotatedModel: models.AnnotatedModel{
						Metadata: convert.DerefHeaderPtr[string](planSubBody.Metadata),
					},
				},
				plan: subscription.PlanRefInput{
					Key:     planSubBody.Plan.Key,
					Version: planSubBody.Plan.Version,
				},
			}, nil
		},
		func(ctx context.Context, request CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
			// TODO: move to new driver
			plan, err := h.SubscrpiptionPlanAdapter.GetVersion(ctx, request.inp.Namespace, request.plan)
			if err != nil {
				return CreateSubscriptionResponse{}, err
			}

			subView, err := h.SubscriptionWorkflowService.CreateFromPlan(ctx, request.inp, plan)
			if err != nil {
				return CreateSubscriptionResponse{}, err
			}

			return MapSubscriptionToAPI(subView.Subscription), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("createSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
