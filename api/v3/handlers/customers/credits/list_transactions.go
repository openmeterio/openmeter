package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	ListCreditTransactionsRequest  = customerbalance.ListCreditTransactionsInput
	ListCreditTransactionsResponse = api.CreditTransactionPaginatedResponse
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

			size := 20
			if args.Params.Page != nil {
				size = lo.FromPtrOr(args.Params.Page.Size, 20)
			}

			if size < 1 {
				err := fmt.Errorf("must be greater than 0")
				return ListCreditTransactionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page.size",
						Reason: "must be greater than 0",
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := customerbalance.ListCreditTransactionsInput{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        args.CustomerID,
				},
				Limit: size,
			}

			if args.Params.Page != nil {
				if args.Params.Page.After != nil {
					after, err := decodeBillingCreditTransactionCursor(*args.Params.Page.After, ns)
					if err != nil {
						return ListCreditTransactionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{
								Field:  "page.after",
								Reason: err.Error(),
								Source: apierrors.InvalidParamSourceQuery,
							},
						})
					}

					req.After = after
				}

				if args.Params.Page.Before != nil {
					before, err := decodeBillingCreditTransactionCursor(*args.Params.Page.Before, ns)
					if err != nil {
						return ListCreditTransactionsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{
								Field:  "page.before",
								Reason: err.Error(),
								Source: apierrors.InvalidParamSourceQuery,
							},
						})
					}

					req.Before = before
				}
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

			// We intentionally expose opaque cursor tokens instead of URI links.
			// This endpoint reuses the shared cursor metadata schema, but still emits
			// opaque token values rather than fully qualified URLs.
			meta := api.CursorMeta{
				Page: api.CursorMetaPage{
					Next:     nullable.NewNullNullable[string](),
					Previous: nullable.NewNullNullable[string](),
					Size:     float32(request.Limit),
				},
			}

			if result.NextCursor != nil {
				next, err := encodeBillingCreditTransactionCursor(*result.NextCursor)
				if err != nil {
					return ListCreditTransactionsResponse{}, fmt.Errorf("encode next cursor: %w", err)
				}
				meta.Page.Next = nullable.NewNullableWithValue(next)
			}

			if result.PreviousCursor != nil {
				previous, err := encodeBillingCreditTransactionCursor(*result.PreviousCursor)
				if err != nil {
					return ListCreditTransactionsResponse{}, fmt.Errorf("encode previous cursor: %w", err)
				}
				meta.Page.Previous = nullable.NewNullableWithValue(previous)
			}

			return api.CreditTransactionPaginatedResponse{
				Data: toAPIBillingCreditTransactions(result.Items),
				Meta: meta,
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
