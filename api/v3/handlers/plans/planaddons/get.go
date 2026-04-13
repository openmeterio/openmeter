package planaddons

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetPlanAddonRequest = planaddon.GetPlanAddonInput
	GetPlanAddonParams  struct {
		PlanID      string
		PlanAddonID string
	}
	GetPlanAddonResponse = api.PlanAddon
	GetPlanAddonHandler  httptransport.HandlerWithArgs[GetPlanAddonRequest, GetPlanAddonResponse, GetPlanAddonParams]
)

func (h *handler) GetPlanAddon() GetPlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetPlanAddonParams) (GetPlanAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPlanAddonRequest{}, err
			}

			return GetPlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID: params.PlanAddonID,
			}, nil
		},
		func(ctx context.Context, request GetPlanAddonRequest) (GetPlanAddonResponse, error) {
			a, err := h.addonService.GetPlanAddon(ctx, request)
			if err != nil {
				return GetPlanAddonResponse{}, err
			}

			return fromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
