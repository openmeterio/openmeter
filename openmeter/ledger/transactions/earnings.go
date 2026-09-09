package transactions

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// RecognizeEarningsFromAttributableAccruedTemplate recognizes up to Amount from accrued
// routes that already have a known cost basis. Unknown-cost accrued balances are skipped.
type RecognizeEarningsFromAttributableAccruedTemplate struct {
	// OriginTracked selects the provenance pool; false is the legacy pool.
	OriginTracked bool
	At            time.Time
	Amount        alpacadecimal.Decimal
	Currency      currencies.CurrencyReference
	// Sources, when provided, are the accrued slices already selected by the
	// caller within its transaction. Nil selects from attributable balances.
	Sources []PostingAmount
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) Validate() error {
	if t.At.IsZero() {
		return fmt.Errorf("at is required")
	}

	if err := ledger.ValidateTransactionAmount(t.Amount); err != nil {
		return fmt.Errorf("amount: %w", err)
	}

	if err := t.Currency.Validate(); err != nil {
		return fmt.Errorf("currency: %w", err)
	}

	if t.Sources != nil {
		total := alpacadecimal.Zero
		for i, source := range t.Sources {
			if source.Address == nil || source.Address.AccountType() != ledger.AccountTypeCustomerAccrued {
				return fmt.Errorf("sources[%d]: customer accrued address is required", i)
			}
			route := source.Address.Route().Route()
			if !route.Currency.Equal(t.Currency) || route.CostBasis == nil || !isCreditBackedAccruedIdentity(source.Identity) {
				return fmt.Errorf("sources[%d]: known-cost credit-backed accrued in the recognition currency is required", i)
			}
			if err := ledger.ValidateTransactionAmount(source.Amount); err != nil {
				return fmt.Errorf("sources[%d]: %w", i, err)
			}
			total = total.Add(source.Amount)
		}
		if !total.Equal(t.Amount) {
			return fmt.Errorf("source total %s does not match recognition amount %s", total, t.Amount)
		}
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
		scope.sourceEntryAmount,
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
		currency:           route.Currency.IdentityKey(),
		costBasisCurrency:  string(lo.FromPtrOr(route.CostBasisCurrency, currencyx.Code(""))),
		taxCode:            lo.FromPtrOr(route.TaxCode, "null"),
		taxBehavior:        string(lo.FromPtrOr(route.TaxBehavior, "null")),
		costBasis:          costBasisKey(route.CostBasis),
		sourceChargeID:     lo.FromPtrOr(identity.SourceChargeID, "null"),
		spendChargeID:      lo.FromPtrOr(identity.SpendChargeID, "null"),
		collectionOriginID: lo.FromPtrOr(identity.CollectionOriginID, "null"),
	}
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) entryRoutePairingKey(entry ledger.Entry) routePairingKey {
	return t.routePairingKey(entry.PostingAddress(), ledger.EntryIdentityParts{
		SourceChargeID:     entry.SourceChargeID(),
		CollectionOriginID: entry.CollectionOriginID(),
		SpendChargeID:      entry.SpendChargeID(),
	})
}

func (t RecognizeEarningsFromAttributableAccruedTemplate) resolve(ctx context.Context, customerID customer.CustomerID, resolvers ResolverDependencies) (ledger.TransactionInput, error) {
	var collections []postingAddressAmount
	if t.Sources == nil {
		var err error
		collections, err = collectFromAttributableCustomerAccrued(ctx, customerID, t.Currency, t.Amount, resolvers, t.OriginTracked, t.At)
		if err != nil {
			return nil, fmt.Errorf("collect from attributable accrued: %w", err)
		}
	} else {
		for _, source := range t.Sources {
			collections = append(collections, postingAddressAmount{address: source.Address, amount: source.Amount, identity: source.Identity})
		}
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
				Currency:          route.Currency,
				CostBasisCurrency: route.CostBasisCurrency,
				TaxCode:           route.TaxCode,
				TaxBehavior:       route.TaxBehavior,
				CostBasis:         route.CostBasis,
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
