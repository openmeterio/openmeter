package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
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
				return ChangeSubscriptionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var workflowInput subscriptionworkflow.ChangeSubscriptionWorkflowInput

			var planInput plansubscription.PlanInput
			var startingPhase *string

			// Try to parse as custom subscription change
			if b, err := body.AsCustomSubscriptionChange(); err == nil {
				// Convert API input to plan creation input using the mapping function
				createPlanInput, err := AsCustomPlanCreateInput(b.CustomPlan, ns)
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to convert custom plan: %w", err)
				}

				// Create the custom plan and set the reference to it in the plan input
				customPlan, err := h.PlanService.CreatePlan(ctx, createPlanInput)
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to create custom plan: %w", err)
				}

				// Publish the custom plan to make it active
				effectiveFrom := createPlanInput.EffectiveFrom
				if effectiveFrom == nil {
					effectiveFrom = lo.ToPtr(clock.Now())
				}
				customPlan, err = h.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
					NamespacedID: customPlan.NamespacedID,
					EffectivePeriod: productcatalog.EffectivePeriod{
						EffectiveFrom: effectiveFrom,
						EffectiveTo:   createPlanInput.EffectiveTo,
					},
				})
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to publish custom plan: %w", err)
				}

				planInput.FromRef(&plansubscription.PlanRefInput{
					Key:     customPlan.Key,
					Version: &customPlan.Version,
				})

				subscriptionTiming, err := MapAPITimingToTiming(b.Timing)
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to map timing: %w", err)
				}

				workflowInput = subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing:      subscriptionTiming,
					Name:        b.CustomPlan.Name,
					Description: b.CustomPlan.Description,
					MetadataModel: models.MetadataModel{
						Metadata: convert.DerefHeaderPtr[string](b.CustomPlan.Metadata),
					},
				}
				// Try to parse as plan subscription change
			} else if b, err := body.AsPlanSubscriptionChange(); err == nil {
				planInput.FromRef(&plansubscription.PlanRefInput{
					Key:     b.Plan.Key,
					Version: b.Plan.Version,
				})

				subscriptionTiming, err := MapAPITimingToTiming(b.Timing)
				if err != nil {
					return ChangeSubscriptionRequest{}, fmt.Errorf("failed to map timing: %w", err)
				}

				startingPhase = b.StartingPhase

				workflowInput = subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing:      subscriptionTiming,
					Name:        lo.FromPtr(b.Name),
					Description: b.Description,
					MetadataModel: models.MetadataModel{
						Metadata: convert.DerefHeaderPtr[string](b.Metadata),
					},
				}
			} else {
				return ChangeSubscriptionRequest{}, models.NewGenericValidationError(fmt.Errorf("invalid request body"))
			}

			return ChangeSubscriptionRequest{
				ID:            models.NamespacedID{Namespace: ns, ID: params.ID},
				PlanInput:     planInput,
				WorkflowInput: workflowInput,
				StartingPhase: startingPhase,
			}, nil
		},
		func(ctx context.Context, request ChangeSubscriptionRequest) (ChangeSubscriptionResponse, error) {
			res, err := h.PlanSubscriptionService.Change(ctx, request)
			if err != nil {
				return ChangeSubscriptionResponse{}, err
			}

			v, err := MapSubscriptionViewToAPI(res.Next)

			return ChangeSubscriptionResponse{
				Current: MapSubscriptionToAPI(res.Current),
				Next:    v,
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
