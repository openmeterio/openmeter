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
	GetPriceRequest  = llmcost.GetPriceInput
	GetPriceResponse = api.LLMCostPrice
	GetPriceHandler  = httptransport.HandlerWithArgs[GetPriceRequest, GetPriceResponse, api.ULID]
)

func (h *handler) GetPrice() GetPriceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, priceID api.ULID) (GetPriceRequest, error) {
			return GetPriceRequest{
				ID: priceID,
			}, nil
		},
		func(ctx context.Context, request GetPriceRequest) (GetPriceResponse, error) {
			price, err := h.service.GetPrice(ctx, request)
			if err != nil {
				return GetPriceResponse{}, err
			}

			return domainPriceToAPI(price), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetPriceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-llm-cost-price"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
