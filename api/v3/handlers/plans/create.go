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
	CreatePlanRequest  = plan.CreatePlanInput
	CreatePlanResponse = api.BillingPlan
	CreatePlanHandler  httptransport.Handler[CreatePlanRequest, CreatePlanResponse]
)

func (h *handler) CreatePlan() CreatePlanHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreatePlanRequest, error) {
			body := api.CreatePlanRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreatePlanRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreatePlanRequest{}, err
			}

			req, err := toCreatePlanInput(ns, body)
			if err != nil {
				return CreatePlanRequest{}, err
			}

			req.IgnoreNonCriticalIssues = true

			return req, nil
		},
		func(ctx context.Context, request CreatePlanRequest) (CreatePlanResponse, error) {
			p, err := h.service.CreatePlan(ctx, request)
			if err != nil {
				return CreatePlanResponse{}, err
			}

			if p == nil {
				return CreatePlanResponse{}, fmt.Errorf("failed to create plan")
			}

			return FromPlan(*p)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreatePlanResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-plan"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
