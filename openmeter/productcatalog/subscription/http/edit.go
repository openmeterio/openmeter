package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	// EditSubscriptionRequest keeps the customizations undecoded: mapping an added item may create
	// Tax Codes for legacy stripe.code values, which has to happen in the same transaction as the edit.
	EditSubscriptionRequest = struct {
		ID             models.NamespacedID
		Customizations []api.SubscriptionEditOperation
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
				Customizations: body.Customizations,
				Timing:         timing,
			}, nil
		},
		func(ctx context.Context, req EditSubscriptionRequest) (EditSubscriptionResponse, error) {
			var resp EditSubscriptionResponse

			err := h.TaxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
				patches := make([]subscription.Patch, 0, len(req.Customizations))
				for idx, patch := range req.Customizations {
					p, err := MapAPISubscriptionEditOperationToPatch(ctx, resolver, req.ID.Namespace, patch)
					if err != nil {
						var resolveErr *productcataloghttp.TaxCodeResolutionError
						if errors.As(err, &resolveErr) {
							return err
						}

						return models.NewGenericValidationError(fmt.Errorf("failed to map patch at idx %d to subscription.Patch: %w", idx, err))
					}

					patches = append(patches, p)
				}

				sub, err := h.SubscriptionWorkflowService.EditRunning(ctx, req.ID, patches, req.Timing)
				if err != nil {
					return err
				}

				resp = MapSubscriptionToAPI(sub.Subscription)

				return nil
			})

			return resp, err
		},
		commonhttp.JSONResponseEncoderWithStatus[EditSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("editSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
