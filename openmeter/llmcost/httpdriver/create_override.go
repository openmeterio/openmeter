package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateOverrideRequest  = llmcost.CreateOverrideInput
	CreateOverrideResponse = api.LLMCostPrice
	CreateOverrideHandler  = httptransport.Handler[CreateOverrideRequest, CreateOverrideResponse]
)

func (h *handler) CreateOverride() CreateOverrideHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateOverrideRequest, error) {
			var body api.LLMCostOverrideCreate
			if err := request.ParseBody(r, &body); err != nil {
				return CreateOverrideRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateOverrideRequest{}, err
			}

			return apiCreateOverrideToDomain(ns, body), nil
		},
		func(ctx context.Context, req CreateOverrideRequest) (CreateOverrideResponse, error) {
			price, err := h.service.CreateOverride(ctx, req)
			if err != nil {
				return CreateOverrideResponse{}, fmt.Errorf("failed to create llm cost override: %w", err)
			}

			return domainPriceToAPI(price), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateOverrideResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-llm-cost-override"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
