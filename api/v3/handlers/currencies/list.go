package currencies

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
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
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCurrenciesRequest{}, err
			}

			page := pagination.NewPage(1, 100)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 100),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCurrenciesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			var orderBy currencies.OrderBy
			var order sortx.Order
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListCurrenciesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				orderBy, err = FromAPICurrencySortField(ctx, sort.Field)
				if err != nil {
					return ListCurrenciesRequest{}, err
				}
				order = sort.Order.ToSortxOrder()
			}

			req := ListCurrenciesRequest{
				Page:      page,
				Namespace: ns,
				OrderBy:   orderBy,
				Order:     order,
			}

			if params.Filter != nil {
				if params.Filter.Type != nil {
					ft := FromAPIBillingCurrencyType(*params.Filter.Type)
					req.FilterType = &ft
				}

				code, err := filters.FromAPIFilterString(params.Filter.Code)
				if err != nil {
					return ListCurrenciesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[code]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Code = code
			}

			return req, nil
		},
		func(ctx context.Context, request ListCurrenciesRequest) (ListCurrenciesResponse, error) {
			result, err := h.currencyService.ListCurrencies(ctx, request)
			if err != nil {
				return ListCurrenciesResponse{}, err
			}

			attrs := []slog.Attr{
				slog.String("operation", "list-currencies"),
				slog.String("namespace", request.Namespace),
				slog.Int("page_number", request.Page.PageNumber),
				slog.Int("page_size", request.Page.PageSize),
				slog.Int("result_count", len(result.Items)),
				slog.Int("total_count", result.TotalCount),
			}
			if request.FilterType != nil {
				attrs = append(attrs, slog.String("filter_type", string(*request.FilterType)))
			}
			slog.LogAttrs(ctx, slog.LevelDebug, "listed currencies", attrs...)

			items := make([]v3.BillingCurrency, 0, len(result.Items))
			for _, c := range result.Items {
				item, err := ToAPIBillingCurrency(c)
				if err != nil {
					return ListCurrenciesResponse{}, err
				}
				items = append(items, item)
			}

			return response.NewPagePaginationResponse(
				items,
				response.PageMetaPage{
					Size:   request.Page.PageSize,
					Number: request.Page.PageNumber,
					Total:  lo.ToPtr(result.TotalCount),
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
