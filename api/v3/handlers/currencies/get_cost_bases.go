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
	ListCostBasesRequest  = currencies.ListCostBasesInput
	ListCostBasesResponse = response.PagePaginationResponse[v3.BillingCostBasis]
	ListCostBasesParams   = v3.ListCostBasesParams
)

// ListCostBasesArgs bundles the path parameter and query parameters for ListCostBases.
type ListCostBasesArgs struct {
	CurrencyID string
	Params     ListCostBasesParams
}

// ListCostBasesHandler is a handler for ListCostBases that accepts two arguments
// (currencyId path param + query params) via a custom With method.
type ListCostBasesHandler interface {
	With(currencyID string, params ListCostBasesParams) httptransport.Handler[ListCostBasesRequest, ListCostBasesResponse]
}

type listCostBasesHandler struct {
	inner httptransport.HandlerWithArgs[ListCostBasesRequest, ListCostBasesResponse, ListCostBasesArgs]
}

func (h listCostBasesHandler) With(currencyID string, params ListCostBasesParams) httptransport.Handler[ListCostBasesRequest, ListCostBasesResponse] {
	return h.inner.With(ListCostBasesArgs{CurrencyID: currencyID, Params: params})
}

func (h *handler) ListCostBases() ListCostBasesHandler {
	inner := httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCostBasesArgs) (ListCostBasesRequest, error) {
			page := pagination.NewPage(1, 20)
			if args.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(args.Params.Page.Number, 1),
					lo.FromPtrOr(args.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCostBasesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			var filterFiatCode *string
			if args.Params.Filter != nil && args.Params.Filter.FiatCode != nil {
				s := *args.Params.Filter.FiatCode
				filterFiatCode = &s
			}

			return ListCostBasesRequest{
				Page:           page,
				CurrencyID:     args.CurrencyID,
				FilterFiatCode: filterFiatCode,
			}, nil
		},
		func(ctx context.Context, req ListCostBasesRequest) (ListCostBasesResponse, error) {
			items, total, err := h.currencyService.ListCostBases(ctx, req)
			if err != nil {
				return ListCostBasesResponse{}, err
			}

			return response.NewPagePaginationResponse(
				lo.Map(items, func(cb currencies.CostBasis, _ int) v3.BillingCostBasis {
					return MapCostBasisToAPI(cb)
				}),
				response.PageMetaPage{
					Size:   req.Page.PageSize,
					Number: req.Page.PageNumber,
					Total:  lo.ToPtr(total),
				},
			), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCostBasesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-cost-bases"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
	return listCostBasesHandler{inner: inner}
}
