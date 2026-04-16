package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCreditTransactionsRequest  = customerbalance.ListCreditTransactionsInput
	ListCreditTransactionsResponse = response.PagePaginationResponse[api.BillingCreditTransaction]
	ListCreditTransactionsParams   struct {
		CustomerID api.ULID
		Params     api.ListCreditTransactionsParams
	}
	ListCreditTransactionsHandler httptransport.HandlerWithArgs[ListCreditTransactionsRequest, ListCreditTransactionsResponse, ListCreditTransactionsParams]
)

func (h *handler) ListCreditTransactions() ListCreditTransactionsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCreditTransactionsParams) (ListCreditTransactionsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCreditTransactionsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if args.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(args.Params.Page.Number, 1),
					lo.FromPtrOr(args.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCreditTransactionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := customerbalance.ListCreditTransactionsInput{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        args.CustomerID,
				},
				Page: page,
			}

			if args.Params.Filter != nil {
				req.Type = fromAPIBillingCreditTransactionType(args.Params.Filter.Type)

				if args.Params.Filter.Currency != nil {
					currency := currencyx.Code(*args.Params.Filter.Currency)
					req.Currency = &currency
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListCreditTransactionsRequest) (ListCreditTransactionsResponse, error) {
			result, err := h.balanceFacade.ListCreditTransactions(ctx, request)
			if err != nil {
				return ListCreditTransactionsResponse{}, fmt.Errorf("list credit transactions: %w", err)
			}

			return response.NewPagePaginationResponse(toAPIBillingCreditTransactions(result.Items), response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCreditTransactionsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-credit-transactions"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
