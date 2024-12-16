package httpdriver

import (
	"context"
	"encoding/json"
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

			// Any transformation function generated by the API will succeed if the body is serializable, so we have to check for the presence of
			// fields to determine what body type we're dealing with
			type testForCustomPlan struct {
				CustomPlan any `json:"customPlan"`
			}

			var t testForCustomPlan

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to marshal request body: %w", err)
			}

			if err := json.Unmarshal(bodyBytes, &t); err != nil {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to unmarshal request body: %w", err)
			}

			if t.CustomPlan != nil {
				// Custom subscription creation
				parsedBody, err := body.AsCustomSubscriptionCreate()
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to decode request body: %w", err)
				}

				req, err := planhttp.AsCreatePlanRequest(parsedBody.CustomPlan, ns)
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to create plan request: %w", err)
				}

				return CreateSubscriptionRequest{
					inp: subscription.CreateSubscriptionWorkflowInput{
						ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
							ActiveFrom:  parsedBody.ActiveFrom,
							Name:        req.Name,        // We map the plan name to the subscription name
							Description: req.Description, // We map the plan description to the subscription description
							AnnotatedModel: models.AnnotatedModel{
								Metadata: req.Metadata, // We map the plan metadata to the subscription metadata
							},
						},
						Namespace:  ns,
						CustomerID: parsedBody.CustomerId,
					},

					plan: &req,
				}, nil
			} else {
				// Plan subscription creation
				parsedBody, err := body.AsPlanSubscriptionCreate()
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to decode request body: %w", err)
				}
				return CreateSubscriptionRequest{
					inp: subscription.CreateSubscriptionWorkflowInput{
						ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
							ActiveFrom:  parsedBody.ActiveFrom,
							Name:        parsedBody.Name,
							Description: parsedBody.Description,
							AnnotatedModel: models.AnnotatedModel{
								Metadata: convert.DerefHeaderPtr[string](parsedBody.Metadata),
							},
						},
						Namespace:  ns,
						CustomerID: parsedBody.CustomerId,
					},
					planRef: &plansubscription.PlanRefInput{
						Key:     parsedBody.Plan.Key,
						Version: parsedBody.Plan.Version,
					},
				}, nil
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
