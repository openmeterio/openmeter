package plans

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetPlanRequest  = plan.GetPlanInput
	GetPlanResponse = api.BillingPlan
	GetPlanParams   = string
	GetPlanHandler  httptransport.HandlerWithArgs[GetPlanRequest, GetPlanResponse, GetPlanParams]
)

func (h *handler) GetPlan() GetPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID GetPlanParams) (GetPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPlanRequest{}, err
			}

			return GetPlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
			}, nil
		},
		func(ctx context.Context, request GetPlanRequest) (GetPlanResponse, error) {
			p, err := h.service.GetPlan(ctx, request)
			if err != nil {
				return GetPlanResponse{}, err
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
