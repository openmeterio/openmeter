package customerscredits

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCreditTransactionsRequest struct {
		Namespace  string
		CustomerID string
		Page       pagination.Page

		TypeFilter     *api.BillingCreditTransactionType
		CurrencyFilter *currencyx.Code
	}
	ListCreditTransactionsResponse = response.PagePaginationResponse[api.BillingCreditTransaction]
	ListCreditTransactionsParams   struct {
		CustomerID api.ULID
		Params     api.ListCreditTransactionsParams
	}
	ListCreditTransactionsHandler httptransport.HandlerWithArgs[ListCreditTransactionsRequest, ListCreditTransactionsResponse, ListCreditTransactionsParams]
	mappedCreditTransaction       struct {
		API      api.BillingCreditTransaction
		Amount   alpacadecimal.Decimal
		Currency currencyx.Code
		Cursor   ledger.TransactionCursor
	}
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

			req := ListCreditTransactionsRequest{
				Namespace:  ns,
				CustomerID: args.CustomerID,
				Page:       page,
			}

			if args.Params.Filter != nil {
				req.TypeFilter = args.Params.Filter.Type

				if args.Params.Filter.Currency != nil {
					currency := currencyx.Code(*args.Params.Filter.Currency)
					req.CurrencyFilter = &currency
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListCreditTransactionsRequest) (ListCreditTransactionsResponse, error) {
			creditMovement, empty := creditMovementFromTypeFilter(request.TypeFilter)
			if empty {
				return emptyCreditTransactionPage(request.Page), nil
			}

			accountID, err := h.customerFBOAccountID(ctx, customer.CustomerID{
				Namespace: request.Namespace,
				ID:        request.CustomerID,
			})
			if err != nil {
				return ListCreditTransactionsResponse{}, fmt.Errorf("resolve customer FBO account: %w", err)
			}

			if accountID == "" {
				return emptyCreditTransactionPage(request.Page), nil
			}

			listIn := ledger.ListTransactionsByPageInput{
				Page:           request.Page,
				Namespace:      request.Namespace,
				AccountIDs:     []string{accountID},
				Currency:       request.CurrencyFilter,
				CreditMovement: creditMovement,
			}

			result, err := h.ledger.ListTransactionsByPage(ctx, listIn)
			if err != nil {
				return ListCreditTransactionsResponse{}, fmt.Errorf("list transactions: %w", err)
			}

			items, err := mapCreditTransactions(result.Items)
			if err != nil {
				return ListCreditTransactionsResponse{}, err
			}

			if len(items) > 0 {
				runningBalance, err := h.customerFBOBalance(ctx, request, items[0].Currency, &items[0].Cursor)
				if err != nil {
					return ListCreditTransactionsResponse{}, fmt.Errorf("get FBO balance after transaction %s: %w", items[0].Cursor.ID.ID, err)
				}

				applyCreditTransactionBalances(items, runningBalance)
			}

			return response.NewPagePaginationResponse(apiCreditTransactions(items), response.PageMetaPage{
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

func emptyCreditTransactionPage(page pagination.Page) ListCreditTransactionsResponse {
	return response.NewPagePaginationResponse([]api.BillingCreditTransaction{}, response.PageMetaPage{
		Size:   page.PageSize,
		Number: page.PageNumber,
		Total:  lo.ToPtr(0),
	})
}

func creditMovementFromTypeFilter(filter *api.BillingCreditTransactionType) (ledger.ListTransactionsCreditMovement, bool) {
	if filter == nil {
		return ledger.ListTransactionsCreditMovementUnspecified, false
	}

	switch *filter {
	case api.BillingCreditTransactionTypeFunded:
		return ledger.ListTransactionsCreditMovementPositive, false
	case api.BillingCreditTransactionTypeConsumed:
		return ledger.ListTransactionsCreditMovementNegative, false
	case api.BillingCreditTransactionTypeAdjusted:
		return ledger.ListTransactionsCreditMovementUnspecified, true
	default:
		return ledger.ListTransactionsCreditMovementUnspecified, false
	}
}

func (h *handler) customerFBOAccountID(ctx context.Context, customerID customer.CustomerID) (string, error) {
	accounts, err := h.accountResolver.GetCustomerAccounts(ctx, customerID)
	if err != nil {
		return "", err
	}

	return fboAccountIDFromCustomerAccounts(accounts), nil
}

func fboAccountIDFromCustomerAccounts(accounts ledger.CustomerAccounts) string {
	if fbo, ok := accounts.FBOAccount.(*ledgeraccount.CustomerFBOAccount); ok {
		return fbo.ID().ID
	}

	return ""
}

func (h *handler) customerFBOBalance(
	ctx context.Context,
	req ListCreditTransactionsRequest,
	currency currencyx.Code,
	after *ledger.TransactionCursor,
) (alpacadecimal.Decimal, error) {
	input := customerbalance.GetBalanceInput{
		CustomerID: customer.CustomerID{
			Namespace: req.Namespace,
			ID:        req.CustomerID,
		},
		Currency: currency,
		After:    after,
	}

	return h.balanceFacade.GetBalance(ctx, input)
}

func applyCreditTransactionBalances(items []mappedCreditTransaction, after alpacadecimal.Decimal) {
	runningBalance := after

	for i := range items {
		items[i].API.AvailableBalance.After = runningBalance.String()
		items[i].API.AvailableBalance.Before = runningBalance.Sub(items[i].Amount).String()
		runningBalance = runningBalance.Sub(items[i].Amount)
	}
}

func mapCreditTransactions(txs []ledger.Transaction) ([]mappedCreditTransaction, error) {
	items := make([]mappedCreditTransaction, 0, len(txs))

	for _, tx := range txs {
		item, err := mapCreditTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("convert ledger transaction %s: %w", tx.ID().ID, err)
		}

		items = append(items, item)
	}

	return items, nil
}

func apiCreditTransactions(items []mappedCreditTransaction) []api.BillingCreditTransaction {
	out := make([]api.BillingCreditTransaction, 0, len(items))
	for _, item := range items {
		out = append(out, item.API)
	}

	return out
}

// mapCreditTransaction maps a ledger.Transaction to the API BillingCreditTransaction type plus its scoped FBO metadata.
func mapCreditTransaction(tx ledger.Transaction) (mappedCreditTransaction, error) {
	entry, err := creditTransactionEntry(tx)
	if err != nil {
		return mappedCreditTransaction{}, err
	}

	createdAt := tx.Cursor().CreatedAt
	amount := entry.Amount()
	currency := entry.PostingAddress().Route().Route().Currency
	txType := creditTransactionType(amount)

	apiTx := api.BillingCreditTransaction{
		Id:        tx.ID().ID,
		CreatedAt: &createdAt,
		BookedAt:  tx.BookedAt(),
		Type:      txType,
		Currency:  api.BillingCurrencyCode(currency),
		Amount:    amount.String(),
		Name:      creditTransactionName(tx),
	}

	labels := creditTransactionLabels(tx)
	if len(labels) > 0 {
		apiLabels := api.Labels(labels)
		apiTx.Labels = &apiLabels
	}

	return mappedCreditTransaction{
		API:      apiTx,
		Amount:   amount,
		Currency: currency,
		Cursor:   tx.Cursor(),
	}, nil
}

func creditTransactionEntry(tx ledger.Transaction) (ledger.Entry, error) {
	for _, entry := range tx.Entries() {
		if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO {
			continue
		}

		return entry, nil
	}

	return nil, fmt.Errorf("no customer FBO entry found in transaction %s", tx.ID().ID)
}

// creditTransactionType determines the type based on the FBO impact sign.
// Positive = funded (balance went up), negative = consumed (balance went down).
func creditTransactionType(fboImpact alpacadecimal.Decimal) api.BillingCreditTransactionType {
	if fboImpact.IsPositive() {
		return api.BillingCreditTransactionTypeFunded
	}

	if fboImpact.IsNegative() {
		return api.BillingCreditTransactionTypeConsumed
	}

	return api.BillingCreditTransactionTypeAdjusted
}

func creditTransactionName(tx ledger.Transaction) string {
	templateName, _ := ledger.TransactionTemplateNameFromAnnotations(tx.Annotations())
	if templateName != "" {
		return templateName
	}

	return "credit_transaction"
}

func creditTransactionLabels(tx ledger.Transaction) map[string]string {
	annotations := tx.Annotations()
	labels := make(map[string]string)

	setLabel := func(key, annotationKey string) {
		value := stringAnnotation(annotations, annotationKey)
		if value != "" {
			labels[key] = value
		}
	}

	setLabel("charge_id", ledger.AnnotationChargeID)
	setLabel("subscription_id", ledger.AnnotationSubscriptionID)
	setLabel("subscription_phase_id", ledger.AnnotationSubscriptionPhaseID)
	setLabel("subscription_item_id", ledger.AnnotationSubscriptionItemID)
	setLabel("feature_id", ledger.AnnotationFeatureID)

	return labels
}

func stringAnnotation(annotations map[string]any, key string) string {
	raw, ok := annotations[key]
	if !ok {
		return ""
	}

	value, ok := raw.(string)
	if !ok {
		return ""
	}

	return value
}
