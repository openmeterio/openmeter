package transactions

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

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

func (t RecognizeEarningsFromAttributableAccruedTemplate) code() TransactionTemplateCode {
	return TemplateCodeRecognizeEarningsFromAttributableAccrued
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

	// Correction owns reversal ordering. Keep it stable so partial corrections
	// are repeatable and independent from the original transaction entry order.
	slices.SortStableFunc(negativeAccruedEntries, compareSubAccountID)
	postings, err := allocateCorrectionLegs(
		negativeAccruedEntries,
		positiveEarningsEntries,
		t.entryRoutePairingKey,
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

func (t RecognizeEarningsFromAttributableAccruedTemplate) routePairingKey(address ledger.PostingAddress, identity ledger.EntryIdentityParts) routePairingKey {
	route := address.Route().Route()

	return routePairingKey{
		currency:       route.Currency,
		taxCode:        lo.FromPtrOr(route.TaxCode, "null"),
		taxBehavior:    string(lo.FromPtrOr(route.TaxBehavior, "null")),
		costBasis:      costBasisKey(route.CostBasis),
		sourceChargeID: lo.FromPtrOr(identity.SourceChargeID, "null"),
		spendChargeID:  lo.FromPtrOr(identity.SpendChargeID, "null"),
	}
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) entryRoutePairingKey(entry ledger.Entry) routePairingKey {
	return t.routePairingKey(entry.PostingAddress(), ledger.EntryIdentityParts{
		SourceChargeID: entry.SourceChargeID(),
		SpendChargeID:  entry.SpendChargeID(),
	})
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
	collections []postingAddressAmount,
) (map[routePairingKey]postingAddressAmount, error) {
	earningsSubAccByKey := make(map[routePairingKey]postingAddressAmount, len(collections))

	for _, collection := range collections {
		key := t.routePairingKey(collection.address, collection.identity)
		current := earningsSubAccByKey[key]
		if current.address == nil {
			// Accrued collection can touch multiple source sub-accounts for the
			// same route and charge provenance. Recognition only needs one earnings
			// destination address per route, but entries must stay split by provenance.
			route := collection.address.Route().Route()
			earnings, err := earningsAccount.GetSubAccountForRoute(ctx, ledger.BusinessRouteParams{
				Currency:    t.Currency,
				TaxCode:     route.TaxCode,
				TaxBehavior: route.TaxBehavior,
				CostBasis:   route.CostBasis,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get earnings sub-account: %w", err)
			}
			current.address = earnings.Address()
			current.identity = collection.identity
		}

		current.amount = current.amount.Add(collection.amount)
		earningsSubAccByKey[key] = current
	}

	return earningsSubAccByKey, nil
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) buildRoutePreservingEarningsEntries(
	collections []postingAddressAmount,
	earningsSubAccByKey map[routePairingKey]postingAddressAmount,
) []*EntryInput {
	entryInputs := make([]*EntryInput, 0, len(collections)*2)

	for _, collection := range collections {
		entryInputs = append(entryInputs, &EntryInput{
			address:  collection.address,
			amount:   collection.amount.Neg(),
			identity: collection.identity,
		})
	}

	// We keep ordering of collections so result is deterministic. It is not needed for correctness.
	creditedKeys := make(map[routePairingKey]struct{}, len(earningsSubAccByKey))
	for _, collection := range collections {
		key := t.routePairingKey(collection.address, collection.identity)
		if _, ok := creditedKeys[key]; ok {
			continue
		}

		earnings := earningsSubAccByKey[key]
		entryInputs = append(entryInputs, &EntryInput{
			address:  earnings.address,
			amount:   earnings.amount,
			identity: earnings.identity,
		})
		creditedKeys[key] = struct{}{}
	}

	return entryInputs
}
