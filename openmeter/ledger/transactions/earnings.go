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

func (t RecognizeEarningsFromAttributableAccruedTemplate) correct(scope CorrectionInput) ([]ledger.TransactionInput, error) {
	type selectedCredit struct {
		earningsAddress ledger.PostingAddress
		accruedAddress  ledger.PostingAddress
		amount          alpacadecimal.Decimal
	}

	// Collect entries from the original recognition transaction:
	// - positive earnings entries (credits to earnings)
	// - negative accrued entries (debits from accrued)
	positiveEarningsEntries := make([]ledger.Entry, 0)
	accruedAddressByCostBasis := make(map[string]ledger.PostingAddress)
	totalAvailable := alpacadecimal.Zero

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeEarnings && entry.Amount().IsPositive():
			positiveEarningsEntries = append(positiveEarningsEntries, entry)
			totalAvailable = totalAvailable.Add(entry.Amount())
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsNegative():
			accruedAddressByCostBasis[costBasisKey(entry.PostingAddress().Route().Route().CostBasis)] = entry.PostingAddress()
		}
	}

	if scope.Amount.GreaterThan(totalAvailable) {
		return nil, fmt.Errorf("earnings correction amount %s exceeds original recognized amount %s", scope.Amount.String(), totalAvailable.String())
	}

	// Consume earnings entries in reverse order (LIFO).
	selected := make([]selectedCredit, 0, len(positiveEarningsEntries))
	remaining := scope.Amount
	for idx := len(positiveEarningsEntries) - 1; idx >= 0 && remaining.IsPositive(); idx-- {
		entry := positiveEarningsEntries[idx]
		amount := entry.Amount()
		if amount.GreaterThan(remaining) {
			amount = remaining
		}

		accruedAddress, ok := accruedAddressByCostBasis[costBasisKey(entry.PostingAddress().Route().Route().CostBasis)]
		if !ok {
			return nil, fmt.Errorf("missing accrued entry for earnings cost basis %s", costBasisKey(entry.PostingAddress().Route().Route().CostBasis))
		}

		selected = append(selected, selectedCredit{
			earningsAddress: entry.PostingAddress(),
			accruedAddress:  accruedAddress,
			amount:          amount,
		})
		remaining = remaining.Sub(amount)
	}

	if remaining.IsPositive() {
		return nil, fmt.Errorf("earnings correction amount %s could not be fully allocated", scope.Amount.String())
	}

	// Build reverse entries: negative earnings (remove), positive accrued (restore).
	// Group accrued restores by sub-account to avoid double-posting.
	accruedAmountsByAddress := make(map[string]selectedCredit)
	entryInputs := make([]*EntryInput, 0, len(selected)*2)

	for _, item := range selected {
		entryInputs = append(entryInputs, &EntryInput{
			address: item.earningsAddress,
			amount:  item.amount.Neg(),
		})

		key := item.accruedAddress.SubAccountID()
		current := accruedAmountsByAddress[key]
		current.accruedAddress = item.accruedAddress
		current.amount = current.amount.Add(item.amount)
		accruedAmountsByAddress[key] = current
	}

	for _, item := range accruedAmountsByAddress {
		entryInputs = append(entryInputs, &EntryInput{
			address: item.accruedAddress,
			amount:  item.amount,
		})
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt:    scope.At,
			entryInputs: entryInputs,
		},
	}, nil
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
