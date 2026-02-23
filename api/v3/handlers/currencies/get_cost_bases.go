package currencies

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetCostBasesByCurrencyIDRequest = currencies.GetCostBasisInput

	GetCostBasesByCurrencyIDResponse = currencies.CostBases
	GetCostBasesByCurrencyIDHandler  httptransport.HandlerWithArgs[GetCostBasesByCurrencyIDRequest, GetCostBasesByCurrencyIDResponse, string]
)

func (h *handler) GetCostBasesByCurrencyID() GetCostBasesByCurrencyIDHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, currencyID string) (GetCostBasesByCurrencyIDRequest, error) {
			return GetCostBasesByCurrencyIDRequest{
				CurrencyID: currencyID,
			}, nil
		},
		func(ctx context.Context, req GetCostBasesByCurrencyIDRequest) (GetCostBasesByCurrencyIDResponse, error) {
			costBases, err := h.currencyService.GetCostBasesByCurrencyID(ctx, req.CurrencyID)
			if err != nil {
				return nil, err
			}
			return costBases, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetCostBasesByCurrencyIDResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("get-cost-bases"))...,
	)
}
