package plans

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdatePlanAddonRequest  = planaddon.UpdatePlanAddonInput
	UpdatePlanAddonResponse = api.PlanAddon
	UpdatePlanAddonParams   = string
	UpdatePlanAddonHandler  httptransport.HandlerWithArgs[UpdatePlanAddonRequest, UpdatePlanAddonResponse, UpdatePlanAddonParams]
)

func (h *handler) UpdatePlanAddon() UpdatePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planAddonID UpdatePlanAddonParams) (UpdatePlanAddonRequest, error) {
			body := api.UpsertPlanAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdatePlanAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanAddonRequest{}, err
			}

			planID := chi.URLParam(r, "planId")

			req := UpdatePlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				ID:            planAddonID,
				PlanID:        planID,
				FromPlanPhase: &body.FromPlanPhase,
				MaxQuantity:   body.MaxQuantity,
			}

			if body.Labels != nil {
				m := labels.ToMetadata(body.Labels)
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

			return fromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
