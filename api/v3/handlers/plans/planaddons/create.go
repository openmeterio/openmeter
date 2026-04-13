package planaddons

import (
	"context"
	"fmt"
	"net/http"

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
	CreatePlanAddonRequest  = planaddon.CreatePlanAddonInput
	CreatePlanAddonResponse = api.PlanAddon
	CreatePlanAddonParams   = string
	CreatePlanAddonHandler  httptransport.HandlerWithArgs[CreatePlanAddonRequest, CreatePlanAddonResponse, CreatePlanAddonParams]
)

func (h *handler) CreatePlanAddon() CreatePlanAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID CreatePlanAddonParams) (CreatePlanAddonRequest, error) {
			body := api.CreatePlanAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreatePlanAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePlanAddonRequest{}, err
			}

			meta, err := labels.ToMetadata(body.Labels)
			if err != nil {
				return CreatePlanAddonRequest{}, err
			}

			return CreatePlanAddonRequest{
				NamespacedModel: models.NamespacedModel{
					Namespace: ns,
				},
				PlanID:        planID,
				AddonID:       body.Addon.Id,
				FromPlanPhase: body.FromPlanPhase,
				MaxQuantity:   body.MaxQuantity,
				Metadata:      meta,
			}, nil
		},
		func(ctx context.Context, request CreatePlanAddonRequest) (CreatePlanAddonResponse, error) {
			a, err := h.addonService.CreatePlanAddon(ctx, request)
			if err != nil {
				return CreatePlanAddonResponse{}, err
			}

			if a == nil {
				return CreatePlanAddonResponse{}, fmt.Errorf("failed to create plan addon")
			}

			return fromPlanAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePlanAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-plan-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
