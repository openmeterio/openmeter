package customerbalance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	pagepagination "github.com/openmeterio/openmeter/pkg/pagination"
)

type CreditTransactionType string

const (
	CreditTransactionTypeFunded   CreditTransactionType = "funded"
	CreditTransactionTypeConsumed CreditTransactionType = "consumed"
	CreditTransactionTypeAdjusted CreditTransactionType = "adjusted"
)

func (t CreditTransactionType) Validate() error {
	switch t {
	case CreditTransactionTypeFunded, CreditTransactionTypeConsumed, CreditTransactionTypeAdjusted:
		return nil
	default:
		return fmt.Errorf("invalid credit transaction type: %s", t)
	}
}

type ListCreditTransactionsInput struct {
	CustomerID customer.CustomerID
	Page       pagepagination.Page

	Type     *CreditTransactionType
	Currency *currencyx.Code
}

func (i ListCreditTransactionsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Page.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("page: %w", err))
	}

	if i.Type != nil {
		if err := i.Type.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("type: %w", err))
		}
	}

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreditTransaction struct {
	ID          models.NamespacedID
	CreatedAt   time.Time
	BookedAt    time.Time
	Type        CreditTransactionType
	Currency    currencyx.Code
	Amount      alpacadecimal.Decimal
	Balance     CreditTransactionBalance
	Name        string
	Annotations models.Annotations
}

type CreditTransactionBalance struct {
	Before alpacadecimal.Decimal
	After  alpacadecimal.Decimal
}

type ListCreditTransactionsResult = pagepagination.Result[CreditTransaction]

func (s *Service) ListCreditTransactions(ctx context.Context, input ListCreditTransactionsInput) (ListCreditTransactionsResult, error) {
	if err := input.Validate(); err != nil {
		return ListCreditTransactionsResult{}, err
	}

	creditMovement, empty, err := ledgerCreditMovement(input.Type)
	if err != nil {
		return ListCreditTransactionsResult{}, err
	}
	if empty {
		return emptyCreditTransactions(input.Page), nil
	}

	accountID, err := s.customerFBOAccountID(ctx, input.CustomerID)
	if err != nil {
		return ListCreditTransactionsResult{}, fmt.Errorf("resolve customer FBO account: %w", err)
	}
	if accountID == "" {
		return emptyCreditTransactions(input.Page), nil
	}

	result, err := s.Ledger.ListTransactionsByPage(ctx, ledger.ListTransactionsByPageInput{
		Page:           input.Page,
		Namespace:      input.CustomerID.Namespace,
		AccountIDs:     []string{accountID},
		Currency:       input.Currency,
		CreditMovement: creditMovement,
	})
	if err != nil {
		return ListCreditTransactionsResult{}, fmt.Errorf("list ledger transactions: %w", err)
	}

	items, err := creditTransactionsFromLedgerTransactions(result.Items)
	if err != nil {
		return ListCreditTransactionsResult{}, err
	}

	if len(items) > 0 {
		runningBalance, err := s.GetBalance(ctx, input.CustomerID, routeFilter(items[0].Currency), lo.ToPtr(result.Items[0].Cursor()))
		if err != nil {
			return ListCreditTransactionsResult{}, fmt.Errorf("get FBO balance after transaction %s: %w", result.Items[0].ID().ID, err)
		}

		applyCreditTransactionBalances(items, runningBalance.Settled())
	}

	return ListCreditTransactionsResult{
		Page:       result.Page,
		TotalCount: result.TotalCount,
		Items:      items,
	}, nil
}

func emptyCreditTransactions(page pagepagination.Page) ListCreditTransactionsResult {
	return ListCreditTransactionsResult{
		Page:       page,
		TotalCount: 0,
		Items:      []CreditTransaction{},
	}
}

func ledgerCreditMovement(txType *CreditTransactionType) (ledger.ListTransactionsCreditMovement, bool, error) {
	if txType == nil {
		return ledger.ListTransactionsCreditMovementUnspecified, false, nil
	}

	switch *txType {
	case CreditTransactionTypeFunded:
		return ledger.ListTransactionsCreditMovementPositive, false, nil
	case CreditTransactionTypeConsumed:
		return ledger.ListTransactionsCreditMovementNegative, false, nil
	case CreditTransactionTypeAdjusted:
		return ledger.ListTransactionsCreditMovementUnspecified, true, nil
	default:
		return ledger.ListTransactionsCreditMovementUnspecified, false, fmt.Errorf("unsupported credit transaction type: %s", *txType)
	}
}

func (s *Service) customerFBOAccountID(ctx context.Context, customerID customer.CustomerID) (string, error) {
	accounts, err := s.AccountResolver.GetCustomerAccounts(ctx, customerID)
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

func creditTransactionsFromLedgerTransactions(txs []ledger.Transaction) ([]CreditTransaction, error) {
	items := make([]CreditTransaction, 0, len(txs))

	for _, tx := range txs {
		item, err := creditTransactionFromLedgerTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("convert ledger transaction %s: %w", tx.ID().ID, err)
		}

		items = append(items, item)
	}

	return items, nil
}

func creditTransactionFromLedgerTransaction(tx ledger.Transaction) (CreditTransaction, error) {
	entry, err := creditTransactionEntry(tx)
	if err != nil {
		return CreditTransaction{}, err
	}

	amount := entry.Amount()

	return CreditTransaction{
		ID:          tx.ID(),
		CreatedAt:   tx.Cursor().CreatedAt,
		BookedAt:    tx.BookedAt(),
		Type:        creditTransactionType(amount),
		Currency:    entry.PostingAddress().Route().Route().Currency,
		Amount:      amount,
		Name:        creditTransactionName(tx),
		Annotations: tx.Annotations(),
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

func applyCreditTransactionBalances(items []CreditTransaction, after alpacadecimal.Decimal) {
	runningBalance := after

	for i := range items {
		items[i].Balance.After = runningBalance
		items[i].Balance.Before = runningBalance.Sub(items[i].Amount)
		runningBalance = runningBalance.Sub(items[i].Amount)
	}
}

func creditTransactionType(fboImpact alpacadecimal.Decimal) CreditTransactionType {
	if fboImpact.IsPositive() {
		return CreditTransactionTypeFunded
	}

	if fboImpact.IsNegative() {
		return CreditTransactionTypeConsumed
	}

	return CreditTransactionTypeAdjusted
}

func creditTransactionName(tx ledger.Transaction) string {
	templateName, _ := ledger.TransactionTemplateNameFromAnnotations(tx.Annotations())
	if templateName != "" {
		return templateName
	}

	return "credit_transaction"
}
