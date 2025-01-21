package httpdriver

import (
	"context"
	"net/http"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListCurrenciesRequest  struct{}
	ListCurrenciesResponse []api.Currency
	ListCurrenciesHandler  httptransport.Handler[ListCurrenciesRequest, ListCurrenciesResponse]
)

func (h *handler) ListCurrencies() ListCurrenciesHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCurrenciesRequest, error) {
			return ListCurrenciesRequest{}, nil
		},
		func(ctx context.Context, request ListCurrenciesRequest) (ListCurrenciesResponse, error) {
			defs := currency.Definitions()

			return lo.Map(lo.Filter(
				defs,
				func(def *currency.Def, _ int) bool {
					// NOTE: this filters out non-iso currencies such as crypto
					return def.ISONumeric != ""
				},
			), func(def *currency.Def, _ int) api.Currency {
				return api.Currency{
					Code:     api.CurrencyCode(def.ISOCode),
					Name:     def.Name,
					Symbol:   def.Symbol,
					Subunits: def.Subunits,
				}
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCurrenciesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listCurrencies"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
