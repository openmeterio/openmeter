package currencies

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCostBasesRequest  = currencies.ListCostBasesInput
	ListCostBasesResponse = response.PagePaginationResponse[v3.BillingCostBasis]
	ListCostBasesParams   = v3.ListCostBasesParams
	ListCostBasesHandler  = httptransport.HandlerWithArgs[ListCostBasesRequest, ListCostBasesResponse, ListCostBasesArgs]
)

type ListCostBasesArgs struct {
	CurrencyID string
	Params     ListCostBasesParams
}

func (h *handler) ListCostBases() ListCostBasesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCostBasesArgs) (ListCostBasesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCostBasesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if args.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(args.Params.Page.Number, 1),
					lo.FromPtrOr(args.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCostBasesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			var filterFiatCode *currencyx.Code
			if args.Params.Filter != nil && args.Params.Filter.FiatCode != nil {
				s := *args.Params.Filter.FiatCode
				filterFiatCode = lo.ToPtr(currencyx.Code(s))
			}

			return ListCostBasesRequest{
				Page:           page,
				Namespace:      ns,
				CurrencyID:     args.CurrencyID,
				FilterFiatCode: filterFiatCode,
			}, nil
		},
		func(ctx context.Context, req ListCostBasesRequest) (ListCostBasesResponse, error) {
			result, err := h.service.ListCostBases(ctx, req)
			if err != nil {
				return ListCostBasesResponse{}, err
			}

			items := lo.Map(result.Items, func(cb currencies.CostBasis, _ int) v3.BillingCostBasis {
				return ToAPIBillingCostBasis(cb)
			})

			return response.NewPagePaginationResponse(
				items,
				response.PageMetaPage{
					Size:   req.Page.PageSize,
					Number: req.Page.PageNumber,
					Total:  lo.ToPtr(result.TotalCount),
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
}
