package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateSubscriptionRequest  = plansubscription.CreateSubscriptionRequest
	CreateSubscriptionResponse = api.Subscription
	CreateSubscriptionHandler  = httptransport.Handler[CreateSubscriptionRequest, CreateSubscriptionResponse]
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

				req, err := CustomPlanToCreatePlanRequest(parsedBody.CustomPlan, ns)
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to create plan request: %w", err)
				}

				plan := plansubscription.PlanInput{}
				plan.FromInput(&req)

				return CreateSubscriptionRequest{
					WorkflowInput: subscription.CreateSubscriptionWorkflowInput{
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
					PlanInput: plan,
				}, nil
			} else {
				// Plan subscription creation
				parsedBody, err := body.AsPlanSubscriptionCreate()
				if err != nil {
					return CreateSubscriptionRequest{}, fmt.Errorf("failed to decode request body: %w", err)
				}

				plan := plansubscription.PlanInput{}
				plan.FromRef(&plansubscription.PlanRefInput{
					Key: parsedBody.Plan.Key,
				})

				return CreateSubscriptionRequest{
					WorkflowInput: subscription.CreateSubscriptionWorkflowInput{
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
					PlanInput: plan,
				}, nil
			}
		},
		func(ctx context.Context, request CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
			res, err := h.PlanSubscriptionService.Create(ctx, request)
			if err != nil {
				return CreateSubscriptionResponse{}, err
			}

			return MapSubscriptionToAPI(res), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("createSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
