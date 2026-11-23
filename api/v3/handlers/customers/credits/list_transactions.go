package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListCreditTransactionsRequest  = customerbalance.ListCreditTransactionsInput
	ListCreditTransactionsResponse = apiv3models.CreditTransactionPaginatedResponse
	ListCreditTransactionsParams   struct {
		CustomerID apiv3models.ULID
		Params     apiv3models.ListCreditTransactionsParams
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

			page, err := request.ParseCursorPage(ctx, args.Params.Page, 20, func(cursor string) (*ledger.TransactionCursor, error) {
				return decodeBillingCreditTransactionCursor(cursor, ns)
			})
			if err != nil {
				return ListCreditTransactionsRequest{}, err
			}

			req := customerbalance.ListCreditTransactionsInput{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        args.CustomerID,
				},
				Limit:  page.Size,
				After:  page.After,
				Before: page.Before,
			}

			if args.Params.Filter != nil {
				req.Type = fromAPIBillingCreditTransactionType(args.Params.Filter.Type)

				if args.Params.Filter.Currency != nil {
					currency := currencyx.Code(*args.Params.Filter.Currency)
					req.Currency = &currency
				}

				featureFilter, err := fromAPICustomerCreditFeatureFilter(args.Params.Filter.FeatureKey)
				if err != nil {
					return ListCreditTransactionsRequest{}, newFeatureKeyFilterBadRequest(ctx, err)
				}

				req.FeatureFilter = featureFilter
			}

			return req, nil
		},
		func(ctx context.Context, request ListCreditTransactionsRequest) (ListCreditTransactionsResponse, error) {
			result, err := h.balanceFacade.ListCreditTransactions(ctx, request)
			if err != nil {
				return ListCreditTransactionsResponse{}, fmt.Errorf("list credit transactions: %w", err)
			}

			var edges response.CursorPageEdges

			if result.NextCursor != nil {
				next, err := encodeBillingCreditTransactionCursor(*result.NextCursor)
				if err != nil {
					return ListCreditTransactionsResponse{}, fmt.Errorf("encode next cursor: %w", err)
				}

				edges.Next = &next
			}

			if result.PreviousCursor != nil {
				previous, err := encodeBillingCreditTransactionCursor(*result.PreviousCursor)
				if err != nil {
					return ListCreditTransactionsResponse{}, fmt.Errorf("encode previous cursor: %w", err)
				}

				edges.Previous = &previous
			}

			return apiv3models.CreditTransactionPaginatedResponse{
				Data: toAPIBillingCreditTransactions(result.Items),
				Meta: response.NewCursorMeta(request.Limit, edges),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCreditTransactionsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-credit-transactions"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
