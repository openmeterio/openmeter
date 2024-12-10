package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// TODO: might need or not need a single interface for using the multiple workflow methods
	CreateSubscriptionRequest = struct {
		inp     subscription.CreateSubscriptionWorkflowInput
		planRef *plansubscription.PlanRefInput
		plan    *plan.CreatePlanInput
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
			custSubBody, errAsCustSub := body.AsCustomSubscriptionCreate()

			// Custom subscription creation
			if errAsPlanSub != nil && errAsCustSub == nil {
				req, err := planhttp.AsCreatePlanRequest(custSubBody.CustomPlan, ns)
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to create plan request: %w", err)
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

					plan: &req,
				}, nil
				// Plan subscription creation
			} else if errAsPlanSub == nil {
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
					planRef: &plansubscription.PlanRefInput{
						Key:     planSubBody.Plan.Key,
						Version: planSubBody.Plan.Version,
					},
				}, nil
			} else {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to decode request body: err1 %w err2 %w", errAsPlanSub, errAsCustSub)
			}
		},
		func(ctx context.Context, request CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
			// First, let's map the input to a Plan
			var plan subscription.Plan

			if request.plan != nil {
				p, err := h.SubscrpiptionPlanAdapter.FromInput(ctx, request.inp.Namespace, *request.plan)
				if err != nil {
					return CreateSubscriptionResponse{}, err
				}

				plan = p
			} else if request.planRef != nil {
				p, err := h.SubscrpiptionPlanAdapter.GetVersion(ctx, request.inp.Namespace, *request.planRef)
				if err != nil {
					return CreateSubscriptionResponse{}, err
				}

				plan = p
			} else {
				return CreateSubscriptionResponse{}, fmt.Errorf("plan or plan reference must be provided")
			}

			// Then let's create the subscription form the plan
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
