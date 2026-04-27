package transactions

import (
	"context"
	"fmt"
	"slices"
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
	// Collect entries from the original recognition transaction:
	// - positive earnings entries (credits to earnings)
	// - negative accrued entries (debits from accrued)
	positiveEarningsEntries := make([]ledger.Entry, 0)
	negativeAccruedEntries := make([]ledger.Entry, 0)

	for _, entry := range scope.OriginalTransaction.Entries() {
		switch {
		case entry.PostingAddress().AccountType() == ledger.AccountTypeEarnings && entry.Amount().IsPositive():
			positiveEarningsEntries = append(positiveEarningsEntries, entry)
		case entry.PostingAddress().AccountType() == ledger.AccountTypeCustomerAccrued && entry.Amount().IsNegative():
			negativeAccruedEntries = append(negativeAccruedEntries, entry)
		}
	}

	slices.SortStableFunc(negativeAccruedEntries, compareSubAccountID)
	postings, err := allocateCorrectionLegs(
		negativeAccruedEntries,
		positiveEarningsEntries,
		t.routePairingKey,
		func(entry ledger.Entry) alpacadecimal.Decimal {
			return entry.Amount().Abs()
		},
		scope.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("allocate earnings correction legs: %w", err)
	}

	return []ledger.TransactionInput{
		&TransactionInput{
			bookedAt:    scope.At,
			entryInputs: mapCorrectionPostingsToEntryInputs(postings),
		},
	}, nil
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) routePairingKey(address ledger.PostingAddress) routePairingKey {
	route := address.Route().Route()

	return routePairingKey{
		currency:  route.Currency,
		costBasis: costBasisKey(route.CostBasis),
	}
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

	earningsSubAccByKey, err := t.resolveEarningsSubAccByRoutePairingKey(ctx, businessAccounts.EarningsAccount, collections)
	if err != nil {
		return nil, err
	}

	return &TransactionInput{
		bookedAt:    t.At,
		entryInputs: t.buildRoutePreservingEarningsEntries(collections, earningsSubAccByKey),
	}, nil
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) resolveEarningsSubAccByRoutePairingKey(
	ctx context.Context,
	earningsAccount ledger.BusinessAccount,
	collections []subAccountAmount,
) (map[routePairingKey]subAccountAmount, error) {
	earningsSubAccByKey := make(map[routePairingKey]subAccountAmount, len(collections))

	for _, collection := range collections {
		key := t.routePairingKey(collection.subAccount.Address())
		current := earningsSubAccByKey[key]
		if current.subAccount == nil {
			earnings, err := earningsAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
				Currency:  t.Currency,
				CostBasis: collection.subAccount.Route().CostBasis,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get earnings sub-account: %w", err)
			}
			current.subAccount = earnings
		}

		current.amount = current.amount.Add(collection.amount)
		earningsSubAccByKey[key] = current
	}

	return earningsSubAccByKey, nil
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) buildRoutePreservingEarningsEntries(
	collections []subAccountAmount,
	earningsSubAccByKey map[routePairingKey]subAccountAmount,
) []*EntryInput {
	entryInputs := make([]*EntryInput, 0, len(collections)*2)
	for _, collection := range collections {
		entryInputs = append(entryInputs, &EntryInput{
			address: collection.subAccount.Address(),
			amount:  collection.amount.Neg(),
		})
	}

	creditedKeys := make(map[routePairingKey]struct{}, len(earningsSubAccByKey))
	for _, collection := range collections {
		key := t.routePairingKey(collection.subAccount.Address())
		if _, ok := creditedKeys[key]; ok {
			continue
		}

		earnings := earningsSubAccByKey[key]
		entryInputs = append(entryInputs, &EntryInput{
			address: earnings.subAccount.Address(),
			amount:  earnings.amount,
		})
		creditedKeys[key] = struct{}{}
	}

	return entryInputs
}
