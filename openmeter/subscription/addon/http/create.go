package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateSubscriptionAddonParams = struct {
		SubscriptionID string
	}
	CreateSubscriptionAddonRequest = struct {
		SubscriptionID models.NamespacedID
		AddonInput     subscriptionworkflow.AddAddonWorkflowInput
	}
	CreateSubscriptionAddonResponse = api.SubscriptionAddon
	CreateSubscriptionAddonHandler  = httptransport.HandlerWithArgs[CreateSubscriptionAddonRequest, CreateSubscriptionAddonResponse, CreateSubscriptionAddonParams]
)

func (h *handler) CreateSubscriptionAddon() CreateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CreateSubscriptionAddonParams) (CreateSubscriptionAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			var body api.CreateSubscriptionAddonJSONRequestBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			addonInput, err := MapCreateSubscriptionAddonRequestToInput(body)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			return CreateSubscriptionAddonRequest{
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        params.SubscriptionID,
				},
				AddonInput: addonInput,
			}, nil
		},
		func(ctx context.Context, req CreateSubscriptionAddonRequest) (CreateSubscriptionAddonResponse, error) {
			var def CreateSubscriptionAddonResponse

			// The add-on's own rate cards are serialized in the response, so reject a unit_config
			// add-on (the v1 shape cannot represent it) before it lands on the subscription. We do
			// NOT reject based on the subscription's plan
			addonToAdd, err := h.AddonService.GetAddon(ctx, addon.GetAddonInput{
				NamespacedID: models.NamespacedID{
					Namespace: req.SubscriptionID.Namespace,
					ID:        req.AddonInput.AddonID,
				},
			})
			if err != nil {
				return def, err
			}

			if addonToAdd.AsProductCatalogAddon().HasUnitConfig() {
				return def, productcatalog.ErrUnitConfigNotRepresentable
			}

			subsAdds, err := h.SubscriptionAddonService.List(ctx, req.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
				SubscriptionID: req.SubscriptionID.ID,
			})
			if err != nil {
				return def, err
			}

			var view subscription.SubscriptionView
			var add subscriptionaddon.SubscriptionAddon

			// If the addon is already present, we'll change the quantity instead as a convenience
			if sAdd, ok := lo.Find(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) bool {
				return subAdd.Addon.ID == req.AddonInput.AddonID
			}); ok {
				view, add, err = h.SubscriptionWorkflowService.ChangeAddonQuantity(ctx, req.SubscriptionID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
					SubscriptionAddonID: sAdd.NamespacedID,
					Quantity:            req.AddonInput.InitialQuantity,
					Timing:              req.AddonInput.Timing,
				})
			} else {
				// Otherwise, we'll create it as per usual
				view, add, err = h.SubscriptionWorkflowService.AddAddon(ctx, req.SubscriptionID, req.AddonInput)
			}

			if err != nil {
				return def, err
			}

			return MapSubscriptionAddonToResponse(view, add)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("createSubscriptionAddon"),
		)...,
	)
}
