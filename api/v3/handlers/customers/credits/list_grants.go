package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type (
	ListCreditGrantsRequest  = creditgrant.ListInput
	ListCreditGrantsResponse = response.PagePaginationResponse[api.BillingCreditGrant]
	ListCreditGrantsParams   struct {
		CustomerID api.ULID
		Params     api.ListCreditGrantsParams
	}
	ListCreditGrantsHandler httptransport.HandlerWithArgs[ListCreditGrantsRequest, ListCreditGrantsResponse, ListCreditGrantsParams]
)

func (h *handler) ListCreditGrants() ListCreditGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCreditGrantsParams) (ListCreditGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCreditGrantsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if args.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(args.Params.Page.Number, 1),
					lo.FromPtrOr(args.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListCreditGrantsRequest{
				Page:       page,
				Namespace:  ns,
				CustomerID: args.CustomerID,
			}

			if args.Params.Filter != nil {
				if args.Params.Filter.FundingMethod != nil {
					fm := convertAPIFundingMethod(*args.Params.Filter.FundingMethod)
					req.FundingMethod = &fm
				}

				if args.Params.Filter.Status != nil {
					status := convertAPIStatusToChargeStatus(*args.Params.Filter.Status)
					req.Status = &status
				}

				if args.Params.Filter.Currency != nil {
					currency := currencyx.Code(*args.Params.Filter.Currency)
					req.Currency = &currency
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListCreditGrantsRequest) (ListCreditGrantsResponse, error) {
			result, err := h.creditGrantService.List(ctx, request)
			if err != nil {
				return ListCreditGrantsResponse{}, fmt.Errorf("list credit grants: %w", err)
			}

			grants, err := slicesx.MapWithErr(result.Items, func(item creditpurchase.Charge) (api.BillingCreditGrant, error) {
				return convertCreditGrant(item)
			})
			if err != nil {
				return ListCreditGrantsResponse{}, fmt.Errorf("converting credit grants: %w", err)
			}

			return response.NewPagePaginationResponse(grants, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCreditGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-credit-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
