package plans

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	UpdatePlanRequest  = plan.UpdatePlanInput
	UpdatePlanResponse = api.BillingPlan
	UpdatePlanParams   = string
	UpdatePlanHandler  httptransport.HandlerWithArgs[UpdatePlanRequest, UpdatePlanResponse, UpdatePlanParams]
)

func (h *handler) UpdatePlan() UpdatePlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID UpdatePlanParams) (UpdatePlanRequest, error) {
			body := api.UpsertPlanRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdatePlanRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdatePlanRequest{}, err
			}

			req, err := FromAPIUpsertPlanRequest(ns, planID, body)
			if err != nil {
				return UpdatePlanRequest{}, err
			}

			req.IgnoreNonCriticalIssues = true

			return req, nil
		},
		func(ctx context.Context, request UpdatePlanRequest) (UpdatePlanResponse, error) {
			p, err := h.service.UpdatePlan(ctx, request)
			if err != nil {
				return UpdatePlanResponse{}, err
			}

			if p == nil {
				return UpdatePlanResponse{}, fmt.Errorf("failed to update plan")
			}

			return ToAPIBillingPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdatePlanResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
