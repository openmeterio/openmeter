package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/customer"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

type (
	CreateSubscriptionRequest = struct {
		plansubscription.CreateSubscriptionRequest
		CustomerRef ref.IDOrKey
	}
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

				timing := subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				}
				if parsedBody.Timing != nil {
					timing, err = MapAPITimingToTiming(*parsedBody.Timing)
					if err != nil {
						return CreateSubscriptionRequest{}, fmt.Errorf("failed to map timing: %w", err)
					}
				}

				ref := ref.IDOrKey{
					ID: lo.FromPtrOr(parsedBody.CustomerId, ""),
				}
				if ref.ID == "" {
					ref.Key = lo.FromPtrOr(parsedBody.CustomerKey, "")
				}

				return CreateSubscriptionRequest{
					CreateSubscriptionRequest: plansubscription.CreateSubscriptionRequest{
						WorkflowInput: subscription.CreateSubscriptionWorkflowInput{
							ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
								Timing:      timing,
								Name:        req.Name,        // We map the plan name to the subscription name
								Description: req.Description, // We map the plan description to the subscription description
								AnnotatedModel: models.AnnotatedModel{
									Metadata: req.Metadata, // We map the plan metadata to the subscription metadata
								},
							},
							Namespace: ns,
						},
						PlanInput: plan,
					},
					CustomerRef: ref,
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

				timing := subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				}
				if parsedBody.Timing != nil {
					timing, err = MapAPITimingToTiming(*parsedBody.Timing)
					if err != nil {
						return CreateSubscriptionRequest{}, fmt.Errorf("failed to map timing: %w", err)
					}
				}

				ref := ref.IDOrKey{
					ID: lo.FromPtrOr(parsedBody.CustomerId, ""),
				}
				if ref.ID == "" {
					ref.Key = lo.FromPtrOr(parsedBody.CustomerKey, "")
				}

				return CreateSubscriptionRequest{
					CreateSubscriptionRequest: plansubscription.CreateSubscriptionRequest{
						WorkflowInput: subscription.CreateSubscriptionWorkflowInput{
							ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
								Timing:      timing,
								Name:        lo.FromPtrOr(parsedBody.Name, ""),
								Description: parsedBody.Description,
								AnnotatedModel: models.AnnotatedModel{
									Metadata: convert.DerefHeaderPtr[string](parsedBody.Metadata),
								},
							},
							Namespace: ns,
						},
						PlanInput: plan,
					},
					CustomerRef: ref,
				}, nil
			}
		},
		func(ctx context.Context, request CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
			// Let's resolve the customer
			// We resolve it in the handler as we don't want to introduce the IdOrKey abstraction to the package internal code
			if err := request.CustomerRef.Validate(); err != nil {
				return CreateSubscriptionResponse{}, &models.GenericUserError{
					Inner: fmt.Errorf("invalid customer ref: %w", err),
				}
			}

			customerID := request.CustomerRef.ID
			if customerID == "" {
				cust, err := h.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
					Key:            lo.ToPtr(request.CustomerRef.Key),
					Namespace:      request.WorkflowInput.Namespace,
					Page:           pagination.NewPage(1, 1),
					IncludeDeleted: false,
				})
				if err != nil {
					return CreateSubscriptionResponse{}, err
				}

				if cust.TotalCount != 1 {
					return CreateSubscriptionResponse{}, &models.GenericConflictError{
						Inner: fmt.Errorf("%d customers found with key %s", cust.TotalCount, request.CustomerRef.Key),
					}
				}

				customerID = cust.Items[0].ID
			}

			request.CreateSubscriptionRequest.WorkflowInput.CustomerID = customerID

			res, err := h.PlanSubscriptionService.Create(ctx, request.CreateSubscriptionRequest)
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
