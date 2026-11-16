package planaddons

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdatePlanAddonRequest = planaddon.UpdatePlanAddonInput
	UpdatePlanAddonParams  struct {
		PlanID      string
		PlanAddonID string
	}
	UpdatePlanAddonResponse = apiv3models.PlanAddon
	UpdatePlanAddonHandler  httptransport.HandlerWithArgs[UpdatePlanAddonRequest, UpdatePlanAddonResponse, UpdatePlanAddonParams]
)

func (h *handler) UpdatePlanAddon() UpdatePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdatePlanAddonParams) (UpdatePlanAddonRequest, error) {
			body := apiv3models.UpsertPlanAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdatePlanAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanAddonRequest{}, err
			}

			req := UpdatePlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID:            params.PlanAddonID,
				PlanID:        params.PlanID,
				FromPlanPhase: &body.FromPlanPhase,
				MaxQuantity:   body.MaxQuantity,
			}

			if body.Labels != nil {
				m, err := labels.ToMetadata(body.Labels)
				if err != nil {
					return UpdatePlanAddonRequest{}, err
				}

				req.Metadata = &m
			}

			return req, nil
		},
		func(ctx context.Context, request UpdatePlanAddonRequest) (UpdatePlanAddonResponse, error) {
			a, err := h.addonService.UpdatePlanAddon(ctx, request)
			if err != nil {
				return UpdatePlanAddonResponse{}, err
			}

			if a == nil {
				return UpdatePlanAddonResponse{}, fmt.Errorf("failed to update plan addon")
			}

			return ToAPIPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
