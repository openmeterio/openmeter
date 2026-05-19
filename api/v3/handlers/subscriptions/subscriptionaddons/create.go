package subscriptionaddons

import (
	"context"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type (
	CreateSubscriptionAddonRequest = struct {
		SubscriptionID models.NamespacedID
		AddonInput     subscriptionworkflow.AddAddonWorkflowInput
	}
	CreateSubscriptionAddonResponse = apiv3.SubscriptionAddon
	CreateSubscriptionAddonParams   = string
	CreateSubscriptionAddonHandler  = httptransport.HandlerWithArgs[CreateSubscriptionAddonRequest, CreateSubscriptionAddonResponse, CreateSubscriptionAddonParams]
)

func (h *handler) CreateSubscriptionAddon() CreateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID CreateSubscriptionAddonParams) (CreateSubscriptionAddonRequest, error) {
			body := apiv3.CreateSubscriptionAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			addonInput, err := mapCreateSubscriptionAddonRequestToInput(body)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			return CreateSubscriptionAddonRequest{
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        subscriptionID,
				},
				AddonInput: addonInput,
			}, nil
		},
		func(ctx context.Context, request CreateSubscriptionAddonRequest) (CreateSubscriptionAddonResponse, error) {
			subsAdds, err := h.addonService.List(ctx, request.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
				SubscriptionID: request.SubscriptionID.ID,
			})
			if err != nil {
				return CreateSubscriptionAddonResponse{}, err
			}

			var added subscriptionaddon.SubscriptionAddon

			// If the addon is already present, we'll change the quantity instead as a convenience
			if sAdd, ok := lo.Find(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) bool {
				return subAdd.Addon.ID == request.AddonInput.AddonID
			}); ok {
				_, added, err = h.SubscriptionWorkflowService.ChangeAddonQuantity(ctx, request.SubscriptionID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
					SubscriptionAddonID: sAdd.NamespacedID,
					Quantity:            request.AddonInput.InitialQuantity,
					Timing:              request.AddonInput.Timing,
				})
			} else {
				// Otherwise, we'll create it as per usual
				_, added, err = h.SubscriptionWorkflowService.AddAddon(ctx, request.SubscriptionID, request.AddonInput)
			}

			if err != nil {
				return CreateSubscriptionAddonResponse{}, err
			}

			return toAPISubscriptionAddon(added)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
