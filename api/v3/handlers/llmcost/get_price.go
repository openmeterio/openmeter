package llmcost

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetPriceRequest  = llmcost.GetPriceInput
	GetPriceResponse = apiv3models.LLMCostPrice
	GetPriceHandler  = httptransport.HandlerWithArgs[GetPriceRequest, GetPriceResponse, apiv3models.ULID]
)

func (h *handler) GetPrice() GetPriceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, priceID apiv3models.ULID) (GetPriceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetPriceRequest{}, err
			}

			return GetPriceRequest{
				ID:        priceID,
				Namespace: ns,
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
