package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	planhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/httpdriver"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	ChangeSubscriptionRequest  = plansubscription.ChangeSubscriptionRequest
	ChangeSubscriptionResponse = api.SubscriptionChangeResponseBody
	ChangeSubscriptionParams   = struct {
		ID string
	}
	ChangeSubscriptionHandler = httptransport.HandlerWithArgs[ChangeSubscriptionRequest, ChangeSubscriptionResponse, ChangeSubscriptionParams]
)

func (h *handler) ChangeSubscription() ChangeSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ChangeSubscriptionParams) (ChangeSubscriptionRequest, error) {
			var body api.ChangeSubscriptionJSONRequestBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			// Any transformation function generated by the API will succeed if the body is serializable, so we have to check for the presence of
			// fields to determine what body type we're dealing with
			type testForCustomPlan struct {
				CustomPlan any `json:"customPlan"`
			}

			var t testForCustomPlan

			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return ChangeSubscriptionRequest{}, fmt.Errorf("failed to marshal request body: %w", err)
			}

			if err := json.Unmarshal(bodyBytes, &t); err != nil {
				return ChangeSubscriptionRequest{}, fmt.Errorf("failed to unmarshal request body: %w", err)
			}

			if t.CustomPlan != nil {
				// Changing to a custom Plan
				parsedBody, err := body.AsCustomSubscriptionChange()
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to parse custom plan: %w", err)
				}

				req, err := planhttp.AsCreatePlanRequest(parsedBody.CustomPlan, ns)
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to create plan request: %w", err)
				}

				planInp := plansubscription.PlanInput{}
				planInp.FromInput(&req)

				return ChangeSubscriptionRequest{
					ID:        models.NamespacedID{Namespace: ns, ID: params.ID},
					PlanInput: planInp,
					WorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
						ActiveFrom:  parsedBody.ActiveFrom,
						Name:        req.Name,
						Description: req.Description,
						AnnotatedModel: models.AnnotatedModel{
							Metadata: req.Metadata,
						},
					},
				}, nil
			} else {
				// Changing to a Plan
				parsedBody, err := body.AsPlanSubscriptionChange()
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to parse plan: %w", err)
				}

				planInp := plansubscription.PlanInput{}
				planInp.FromRef(&plansubscription.PlanRefInput{
					Key:     parsedBody.Plan.Key,
					Version: parsedBody.Plan.Version,
				})

				return ChangeSubscriptionRequest{
					ID:        models.NamespacedID{Namespace: ns, ID: params.ID},
					PlanInput: planInp,
					WorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
						ActiveFrom: parsedBody.ActiveFrom,
						AnnotatedModel: models.AnnotatedModel{
							Metadata: convert.DerefHeaderPtr[string](parsedBody.Metadata),
						},
						Name:        parsedBody.Name,
						Description: parsedBody.Description,
					},
				}, nil
			}
		},
		func(ctx context.Context, request ChangeSubscriptionRequest) (ChangeSubscriptionResponse, error) {
			res, err := h.PlanSubscriptionService.Change(ctx, request)
			if err != nil {
				return ChangeSubscriptionResponse{}, err
			}

			v, err := MapSubscriptionViewToAPI(res.New)

			return ChangeSubscriptionResponse{
				Current: MapSubscriptionToAPI(res.Current),
				New:     v,
			}, err
		},
		commonhttp.JSONResponseEncoderWithStatus[ChangeSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("changeSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
