package plans

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeletePlanRequest  = plan.DeletePlanInput
	DeletePlanResponse = any
	DeletePlanParams   = string
	DeletePlanHandler  httptransport.HandlerWithArgs[DeletePlanRequest, DeletePlanResponse, DeletePlanParams]
)

func (h *handler) DeletePlan() DeletePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID DeletePlanParams) (DeletePlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeletePlanRequest{}, err
			}

			return DeletePlanRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        planID,
				},
			}, nil
		},
		func(ctx context.Context, request DeletePlanRequest) (DeletePlanResponse, error) {
			if err := h.service.DeletePlan(ctx, request); err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeletePlanResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("delete-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
