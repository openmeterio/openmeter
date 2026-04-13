package planaddons

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeletePlanAddonRequest = planaddon.DeletePlanAddonInput
	DeletePlanAddonParams  struct {
		PlanID      string
		PlanAddonID string
	}
	DeletePlanAddonResponse = any
	DeletePlanAddonHandler  httptransport.HandlerWithArgs[DeletePlanAddonRequest, DeletePlanAddonResponse, DeletePlanAddonParams]
)

func (h *handler) DeletePlanAddon() DeletePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeletePlanAddonParams) (DeletePlanAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePlanAddonRequest{}, err
			}

			return DeletePlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID:     params.PlanAddonID,
				PlanID: params.PlanID,
			}, nil
		},
		func(ctx context.Context, request DeletePlanAddonRequest) (DeletePlanAddonResponse, error) {
			if err := h.addonService.DeletePlanAddon(ctx, request); err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePlanAddonResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
