package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	EditSubscriptionRequest = struct {
		ID             models.NamespacedID
		Customizations []subscription.Patch
		Timing         subscription.Timing
	}
	EditSubscriptionResponse = api.Subscription
	EditSubscriptionParams   = struct {
		ID string
	}
	EditSubscriptionHandler = httptransport.HandlerWithArgs[EditSubscriptionRequest, EditSubscriptionResponse, EditSubscriptionParams]
)

func (h *handler) EditSubscription() EditSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params EditSubscriptionParams) (EditSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return EditSubscriptionRequest{}, err
			}

			var body api.EditSubscriptionJSONRequestBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return EditSubscriptionRequest{}, err
			}

			if len(body.Customizations) == 0 {
				return EditSubscriptionRequest{}, models.NewGenericValidationError(fmt.Errorf("missing customizations"))
			}

			patches := make([]subscription.Patch, 0, len(body.Customizations))
			for idx, patch := range body.Customizations {
				p, err := MapAPISubscriptionEditOperationToPatch(patch)
				if err != nil {
					return EditSubscriptionRequest{}, models.NewGenericValidationError(fmt.Errorf("failed to map patch at idx %d to subscription.Patch: %w", idx, err))
				}

				patches = append(patches, p)
			}

			var timing subscription.Timing

			if body.Timing != nil {
				timing, err = MapAPITimingToTiming(*body.Timing)
				if err != nil {
					return EditSubscriptionRequest{}, models.NewGenericValidationError(fmt.Errorf("failed to map timing to subscription.Timing: %w", err))
				}
			} else {
				timing = subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				}
			}

			return EditSubscriptionRequest{
				ID:             models.NamespacedID{Namespace: ns, ID: params.ID},
				Customizations: patches,
				Timing:         timing,
			}, nil
		},
		func(ctx context.Context, req EditSubscriptionRequest) (EditSubscriptionResponse, error) {
			sub, err := h.SubscriptionWorkflowService.EditRunning(ctx, req.ID, req.Customizations, req.Timing)
			if err != nil {
				return EditSubscriptionResponse{}, err
			}

			return MapSubscriptionToAPI(sub.Subscription), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[EditSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("editSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
