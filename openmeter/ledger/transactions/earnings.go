package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// RecognizeEarningsFromAttributableAccruedTemplate recognizes up to Amount from accrued
// routes that already have a known cost basis. Unknown-cost accrued balances are skipped.
type RecognizeEarningsFromAttributableAccruedTemplate struct {
	At       time.Time
	Amount   alpacadecimal.Decimal
	Currency currencyx.Code
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := ledger.ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	return nil
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) typeGuard() guard {
	return true
}

var _ CustomerTransactionTemplate = (RecognizeEarningsFromAttributableAccruedTemplate{})

func (t RecognizeEarningsFromAttributableAccruedTemplate) correct(context.Context, CorrectionInput, ResolverDependencies) ([]ledger.TransactionInput, error) {
	return nil, templateCorrectionNotImplemented(templateName(t))
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	collections, err := collectFromAttributableCustomerAccrued(ctx, customerID, t.Currency, t.Amount, resolvers)
	if err != nil {
		return nil, fmt.Errorf("collect from attributable accrued: %w", err)
	}
	if len(collections) == 0 {
		return nil, nil
	}

	businessAccounts, err := resolvers.AccountService.GetBusinessAccounts(ctx, customerID.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get business accounts: %w", err)
	}

	entryInputs := make([]*EntryInput, 0, len(collections)*2)
	for _, collection := range collections {
		earnings, err := businessAccounts.EarningsAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
			Currency:  t.Currency,
			CostBasis: collection.subAccount.Route().CostBasis,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get earnings sub-account: %w", err)
		}

		entryInputs = append(entryInputs, &EntryInput{
			address: collection.subAccount.Address(),
			amount:  collection.amount.Neg(),
		}, &EntryInput{
			address: earnings.Address(),
			amount:  collection.amount,
		})
	}

	return &TransactionInput{
		bookedAt:    t.At,
		entryInputs: entryInputs,
	}, nil
}
