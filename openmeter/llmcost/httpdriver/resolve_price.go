package httpdriver

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ResolvePriceRequest  = llmcost.ResolvePriceInput
	ResolvePriceResponse = api.LLMCostPrice
	ResolvePriceParams   = api.ResolveLlmCostPriceParams
	ResolvePriceHandler  = httptransport.HandlerWithArgs[ResolvePriceRequest, ResolvePriceResponse, ResolvePriceParams]
)

func (h *handler) ResolvePrice() ResolvePriceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ResolvePriceParams) (ResolvePriceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ResolvePriceRequest{}, err
			}

			return ResolvePriceRequest{
				Namespace: ns,
				Provider:  llmcost.Provider(params.Provider),
				ModelID:   params.ModelId,
				At:        params.At,
			}, nil
		},
		func(ctx context.Context, request ResolvePriceRequest) (ResolvePriceResponse, error) {
			price, err := h.service.ResolvePrice(ctx, request)
			if err != nil {
				return ResolvePriceResponse{}, err
			}

			return domainPriceToAPI(price), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ResolvePriceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("resolve-llm-cost-price"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
