package currencies

import (
	"context"
	"net/http"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/samber/lo"
)

type (
	ListCurrenciesRequest  struct{}
	ListCurrenciesResponse []v3.BillingCurrency
	ListCurrenciesHandler  httptransport.Handler[ListCurrenciesRequest, ListCurrenciesResponse]
)

func (h *handler) ListCurrencies() ListCurrenciesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCurrenciesRequest, error) {
			return ListCurrenciesRequest{}, nil
		},
		func(ctx context.Context, request ListCurrenciesRequest) (ListCurrenciesResponse, error) {
			defs, err := h.currencyService.ListCurrencies(ctx)
			if err != nil {
				return nil, err
			}

			return lo.Map(defs, func(def currencies.Currency, _ int) v3.BillingCurrency {
				return MapCurrencyToAPI(def)
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCurrenciesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCurrencies"),
		)...,
	)
}
