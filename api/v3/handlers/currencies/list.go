package currencies

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCurrenciesRequest  = currencies.ListCurrenciesInput
	ListCurrenciesResponse = response.PagePaginationResponse[v3.BillingCurrency]
	ListCurrenciesParams   = v3.ListCurrenciesParams
	ListCurrenciesHandler  httptransport.HandlerWithArgs[ListCurrenciesRequest, ListCurrenciesResponse, ListCurrenciesParams]
)

func (h *handler) ListCurrencies() ListCurrenciesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCurrenciesParams) (ListCurrenciesRequest, error) {
			page := pagination.NewPage(1, 100)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 100),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCurrenciesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			var filterType *currencies.CurrencyType
			if params.Filter != nil && params.Filter.Type != nil {
				ft := MapCurrencyTypeFromAPI(*params.Filter.Type)
				filterType = &ft
			}

			return ListCurrenciesRequest{
				Page:       page,
				FilterType: filterType,
			}, nil
		},
		func(ctx context.Context, request ListCurrenciesRequest) (ListCurrenciesResponse, error) {
			items, total, err := h.currencyService.ListCurrencies(ctx, request)
			if err != nil {
				return ListCurrenciesResponse{}, err
			}

			return response.NewPagePaginationResponse(
				lo.Map(items, func(def currencies.Currency, _ int) v3.BillingCurrency {
					return MapCurrencyToAPI(def)
				}),
				response.PageMetaPage{
					Size:   request.Page.PageSize,
					Number: request.Page.PageNumber,
					Total:  lo.ToPtr(total),
				},
			), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCurrenciesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-currencies"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
