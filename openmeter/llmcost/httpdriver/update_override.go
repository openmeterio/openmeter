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
	UpdateOverrideRequest  = llmcost.UpdateOverrideInput
	UpdateOverrideResponse = api.LLMCostPrice
	UpdateOverrideHandler  = httptransport.HandlerWithArgs[UpdateOverrideRequest, UpdateOverrideResponse, api.ULID]
)

func (h *handler) UpdateOverride() UpdateOverrideHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, overrideID api.ULID) (UpdateOverrideRequest, error) {
			var body api.LLMCostOverrideUpdate
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateOverrideRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateOverrideRequest{}, err
			}

			return apiUpdateOverrideToDomain(ns, overrideID, body), nil
		},
		func(ctx context.Context, req UpdateOverrideRequest) (UpdateOverrideResponse, error) {
			price, err := h.service.UpdateOverride(ctx, req)
			if err != nil {
				return UpdateOverrideResponse{}, fmt.Errorf("failed to update llm cost override: %w", err)
			}

			return domainPriceToAPI(price), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateOverrideResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-llm-cost-override"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
