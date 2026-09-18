package advance

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// advanceReceivableBalances queries balance buckets rather than sub-account
// balances because one receivable sub-account can contain multiple spend-charge
// provenance buckets. Backfill needs those buckets split so each translated
// entry preserves the spend charge that created the advance.
func (p backfillPlanner) advanceReceivableBalances(ctx context.Context, receivableAccountID models.NamespacedID, currency currencies.CurrencyReference) ([]advanceReceivableBalance, error) {
	openStatus := ledger.TransactionAuthorizationStatusOpen

	buckets, err := p.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: receivableAccountID.Namespace,
		Filters: ledger.Filters{
			AccountID: &receivableAccountID.ID,
			Provenance: ledger.ProvenanceFilter{
				SourceChargeID: mo.Some[*string](nil),
				SpendChargeID:  mo.None[*string](),
			},
			Route: ledger.RouteFilter{
				Currency:                       currency,
				CostBasis:                      mo.Some[*alpacadecimal.Decimal](nil),
				TransactionAuthorizationStatus: &openStatus,
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID, ledger.BalanceBucketGroupByCollectionOriginID},
	})
	if err != nil {
		return nil, err
	}

	out := make([]advanceReceivableBalance, 0, len(buckets))

	for _, bucket := range buckets {
		if bucket.SettledAmount.IsZero() {
			continue
		}

		spendChargeID := bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]
		out = append(out, advanceReceivableBalance{
			address:            bucket.Address,
			spendChargeID:      spendChargeID,
			spendChargeKey:     advanceSpendKey(spendChargeID, bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID]),
			collectionOriginID: bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID],
			amount:             bucket.SettledAmount,
		})
	}

	return out, nil
}

// unattributedAccruedBalances returns source-less, nil-cost-basis accrued value
// that can be attributed to a creditpurchase. It groups by spend charge and tax
// dimensions because cost-basis backfill must preserve both dimensions when it
// moves accrued value into the purchased source bucket.
func (p backfillPlanner) unattributedAccruedBalances(ctx context.Context, accruedAccount ledger.CustomerAccruedAccount, currency currencies.CurrencyReference) ([]unattributedAccruedBalance, error) {
	buckets, err := p.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: accruedAccount.ID().Namespace,
		Filters: ledger.Filters{
			AccountID: lo.ToPtr(accruedAccount.ID().ID),
			Provenance: ledger.ProvenanceFilter{
				SourceChargeID: mo.Some[*string](nil),
			},
			Route: ledger.RouteFilter{
				Currency:  currency,
				CostBasis: mo.Some[*alpacadecimal.Decimal](nil),
			},
		},
		GroupBy: []string{ledger.BalanceBucketGroupBySpendChargeID, ledger.BalanceBucketGroupByCollectionOriginID},
	})
	if err != nil {
		return nil, fmt.Errorf("list unattributed accrued balances: %w", err)
	}

	balancesByKey := make(map[accruedBackfillBucketKey]unattributedAccruedBalance, len(buckets))
	keys := make([]accruedBackfillBucketKey, 0, len(buckets))

	for _, bucket := range buckets {
		balance := bucket.SettledAmount
		if !balance.IsPositive() {
			continue
		}

		route := bucket.Address.Route().Route()
		spendChargeID := bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID]

		key := accruedBackfillBucketKey{
			spendChargeID:   advanceSpendKey(spendChargeID, bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID]),
			taxDimensionKey: taxDimensionRouteKey(route),
		}

		if _, ok := balancesByKey[key]; !ok {
			keys = append(keys, key)
			balancesByKey[key] = unattributedAccruedBalance{
				key:                          key,
				collectionOriginID:           bucket.GroupByValues[ledger.BalanceBucketGroupByCollectionOriginID],
				oldestMatchingEntryCreatedAt: bucket.OldestMatchingEntryCreatedAt,
				taxCode:                      route.TaxCode,
				taxBehavior:                  route.TaxBehavior,
			}
		}

		current := balancesByKey[key]
		current.amount = current.amount.Add(balance)
		balancesByKey[key] = current
	}

	slices.SortFunc(keys, cmpx.Compare[accruedBackfillBucketKey])

	return lo.Map(keys, func(key accruedBackfillBucketKey, _ int) unattributedAccruedBalance {
		return balancesByKey[key]
	}), nil
}

// unattributedAccruedBalance is source-less accrued value available for
// creditpurchase backfill. It is keyed by the dimensions that must be preserved
// during cost-basis translation: tax treatment and spend charge provenance.
type unattributedAccruedBalance struct {
	collectionOriginID           *string
	oldestMatchingEntryCreatedAt time.Time
	key                          accruedBackfillBucketKey
	taxCode                      *string
	taxBehavior                  *ledger.TaxBehavior
	amount                       alpacadecimal.Decimal
}

// advanceReceivableBalance is an open source-less receivable bucket that may be
// attributed to a later creditpurchase. The posting address preserves route
// dimensions, while spendChargeKey identifies which spend created the advance.
type advanceReceivableBalance struct {
	collectionOriginID *string
	address            ledger.PostingAddress
	spendChargeID      *string
	// spendChargeKey combines spend charge and collection origin for matching.
	spendChargeKey string
	amount         alpacadecimal.Decimal
	remaining      alpacadecimal.Decimal
}

func (b advanceReceivableBalance) Compare(other advanceReceivableBalance) int {
	if c := cmp.Compare(b.spendChargeKey, other.spendChargeKey); c != 0 {
		return c
	}

	if c := cmpx.Compare(postingAddressRouteKeyFromAddress(b.address), postingAddressRouteKeyFromAddress(other.address)); c != 0 {
		return c
	}

	return cmp.Compare(b.address.SubAccountID(), other.address.SubAccountID())
}

// advanceReceivableBuckets is the mutable allocation state for source-less
// advance receivable. Matching happens by spend charge when it exists; legacy
// rows have no spend charge, so each route bucket remains separate inside the
// same spend group and is consumed in deterministic route order.
type advanceReceivableBuckets struct {
	requiredFeatures []string
	bySpendChargeID  map[string][]advanceReceivableBalance
}

// availableForSpend applies the same feature restriction as consume, so a
// lineage occurrence cannot borrow capacity from another receivable route.
func (b *advanceReceivableBuckets) availableForSpend(spendChargeID string) alpacadecimal.Decimal {
	available := alpacadecimal.Zero

	for _, balance := range b.bySpendChargeID[spendChargeID] {
		if !slices.Equal(b.requiredFeatures, balance.address.Route().Route().Features) {
			continue
		}

		available = available.Add(balance.remaining)
	}

	return available
}

// consume removes up to amount from the concrete receivable buckets for one
// spend key and the current occurrence's feature route.
func (b *advanceReceivableBuckets) consume(spendChargeID string, amount alpacadecimal.Decimal, attributionFor func(advanceReceivableBalance, alpacadecimal.Decimal) advanceAttribution) ([]advanceAttribution, alpacadecimal.Decimal) {
	remainingAmount := amount
	advanceReceivables := b.bySpendChargeID[spendChargeID]
	attributions := make([]advanceAttribution, 0, len(advanceReceivables))
	consumedAmount := alpacadecimal.Zero

	for i := range advanceReceivables {
		if !remainingAmount.IsPositive() {
			break
		}

		advanceReceivable := advanceReceivables[i]
		if !slices.Equal(b.requiredFeatures, advanceReceivable.address.Route().Route().Features) {
			continue
		}

		if !advanceReceivable.remaining.IsPositive() {
			continue
		}

		advanceAmount := advanceReceivable.remaining
		if advanceAmount.GreaterThan(remainingAmount) {
			advanceAmount = remainingAmount
		}

		advanceReceivables[i].remaining = advanceReceivable.remaining.Sub(advanceAmount)
		remainingAmount = remainingAmount.Sub(advanceAmount)
		consumedAmount = consumedAmount.Add(advanceAmount)
		attributions = append(attributions, attributionFor(advanceReceivable, advanceAmount))
	}

	b.bySpendChargeID[spendChargeID] = advanceReceivables

	return attributions, consumedAmount
}

// attributeRemaining uses the purchase remainder against eligible receivable
// before issuing new credit. It preserves each original spend/feature route,
// but does not imply that accrued was translated or a lineage segment funded.
func (b *advanceReceivableBuckets) attributeRemaining(amount alpacadecimal.Decimal) []advanceAttribution {
	var attributions []advanceAttribution

	for _, spendKey := range slices.Sorted(maps.Keys(b.bySpendChargeID)) {
		balances := b.bySpendChargeID[spendKey]

		for i := range balances {
			if !amount.IsPositive() {
				return attributions
			}

			balance := &balances[i]
			if !balance.remaining.IsPositive() {
				continue
			}

			attributed := legacylineage.MinDecimal(amount, balance.remaining)
			attributions = append(attributions, advanceAttribution{
				advanceFeatures:    balance.address.Route().Route().Features,
				spendChargeID:      balance.spendChargeID,
				collectionOriginID: balance.collectionOriginID,
				advanceAmount:      attributed,
			})
			balance.remaining = balance.remaining.Sub(attributed)
			amount = amount.Sub(attributed)
		}
	}

	return attributions
}

// newAdvanceReceivableBuckets selects open source-less advance receivable that
// this creditpurchase is allowed to backfill. Buckets are grouped by spend charge
// for provenance matching, while the original route buckets remain ordered inside
// the group so legacy nil-spend entries cannot overwrite each other.
func newAdvanceReceivableBuckets(advanceReceivables []advanceReceivableBalance, creditFeatures []string) advanceReceivableBuckets {
	buckets := advanceReceivableBuckets{
		bySpendChargeID: make(map[string][]advanceReceivableBalance, len(advanceReceivables)),
	}

	for _, advanceReceivable := range advanceReceivables {
		advanceFeatures := advanceReceivable.address.Route().Route().Features
		if !legacylineage.FeatureFiltersMatchAdvance(creditFeatures, advanceFeatures) {
			continue
		}

		if !advanceReceivable.amount.IsNegative() {
			continue
		}

		advanceReceivable.remaining = advanceReceivable.amount.Neg()
		buckets.bySpendChargeID[advanceReceivable.spendChargeKey] = append(buckets.bySpendChargeID[advanceReceivable.spendChargeKey], advanceReceivable)
	}

	return buckets
}

// postingAddressRouteKey is the comparable subset of a posting address needed
// for deterministic helper ordering. It is not a balance key; balance matching
// is handled by the attribution keys above.
type postingAddressRouteKey struct {
	routingKey   string
	subAccountID string
}

func (k postingAddressRouteKey) Compare(other postingAddressRouteKey) int {
	if c := cmp.Compare(k.routingKey, other.routingKey); c != 0 {
		return c
	}

	return cmp.Compare(k.subAccountID, other.subAccountID)
}

// postingAddressRouteKeyFromAddress extracts the stable route/sub-account
// ordering fields from hydrated balance bucket addresses.
func postingAddressRouteKeyFromAddress(address ledger.PostingAddress) postingAddressRouteKey {
	return postingAddressRouteKey{
		routingKey:   address.Route().RoutingKey().Value(),
		subAccountID: address.SubAccountID(),
	}
}

// advanceSpendKey prevents two runs of the same charge from sharing attribution.
func advanceSpendKey(spend, origin *string) string {
	if origin == nil {
		return lo.FromPtrOr(spend, "null")
	}

	return lo.FromPtrOr(spend, "null") + ":" + *origin
}
