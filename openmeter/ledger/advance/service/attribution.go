package service

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/cmpx"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// advanceAttribution keeps the amounts and original routes needed to attribute
// receivable and accrued to purchased credit without changing spend provenance.
type advanceAttribution struct {
	collectionOriginID *string
	taxCode            *string
	taxBehavior        *ledger.TaxBehavior
	advanceFeatures    []string
	spendChargeID      *string
	advanceAmount      alpacadecimal.Decimal
	accruedAmount      alpacadecimal.Decimal
}

// canMergeInto preserves the tax route of accrued. A receivable-only remainder
// can join an existing attribution regardless of tax, since it adds no accrued
// value. Duplicate receivable entries would make corrections ambiguous.
func (a advanceAttribution) canMergeInto(other advanceAttribution) bool {
	if lo.FromPtr(a.collectionOriginID) != lo.FromPtr(other.collectionOriginID) {
		return false
	}

	if lo.FromPtr(a.spendChargeID) != lo.FromPtr(other.spendChargeID) || !slices.Equal(a.advanceFeatures, other.advanceFeatures) {
		return false
	}

	return !a.accruedAmount.IsPositive() ||
		(lo.FromPtr(a.taxCode) == lo.FromPtr(other.taxCode) && lo.FromPtr(a.taxBehavior) == lo.FromPtr(other.taxBehavior))
}

// taxDimensionKey keeps tax-bearing accrued balances separate because
// backfilling credit source/cost basis must not merge taxable and non-taxable
// accrued buckets.
type taxDimensionKey struct {
	taxCode     string
	taxBehavior string
}

func (k taxDimensionKey) Compare(other taxDimensionKey) int {
	if c := cmp.Compare(k.taxCode, other.taxCode); c != 0 {
		return c
	}

	return cmp.Compare(k.taxBehavior, other.taxBehavior)
}

// accruedBackfillBucketKey adds the accrued dimensions that must remain split
// during cost-basis translation after a receivable bucket has matched by spend.
type accruedBackfillBucketKey struct {
	spendChargeID string
	taxDimensionKey
}

func (k accruedBackfillBucketKey) Compare(other accruedBackfillBucketKey) int {
	if c := cmp.Compare(k.spendChargeID, other.spendChargeID); c != 0 {
		return c
	}

	return cmpx.Compare(k.taxDimensionKey, other.taxDimensionKey)
}

// taxDimensionRouteKey converts nullable tax route fields into comparable
// sentinel values. The sentinel is only for matching; actual posting still uses
// the route values from the hydrated balance bucket.
func taxDimensionRouteKey(route ledger.Route) taxDimensionKey {
	return taxDimensionKey{
		taxCode:     lo.FromPtrOr(route.TaxCode, "null"),
		taxBehavior: string(lo.FromPtrOr(route.TaxBehavior, "null")),
	}
}

// allocateAccruedBackedAdvanceAttributions consumes receivable buckets for
// accrued value that can also be moved into the new creditpurchase cost basis.
// The accrued allocation chooses spend/tax buckets; this function maps each
// allocated spend bucket back onto the concrete receivable route buckets that
// must be cleared.
func allocateAccruedBackedAdvanceAttributions(
	accruedAllocations []currencyx.AmountAllocation[accruedBackfillBucketKey],
	unattributedAccrued []unattributedAccruedBalance,
	receivableBuckets *advanceReceivableBuckets,
) ([]advanceAttribution, error) {
	attributions := make([]advanceAttribution, 0, len(accruedAllocations))

	for _, allocation := range accruedAllocations {
		for i := range unattributedAccrued {
			if unattributedAccrued[i].key != allocation.Key {
				continue
			}

			allocated, consumed := receivableBuckets.consume(allocation.Key.spendChargeID, allocation.Amount, func(advanceReceivable advanceReceivableBalance, amount alpacadecimal.Decimal) advanceAttribution {
				return advanceAttribution{
					taxCode:            unattributedAccrued[i].taxCode,
					taxBehavior:        unattributedAccrued[i].taxBehavior,
					advanceFeatures:    advanceReceivable.address.Route().Route().Features,
					spendChargeID:      advanceReceivable.spendChargeID,
					collectionOriginID: advanceReceivable.collectionOriginID,
					advanceAmount:      amount,
					accruedAmount:      amount,
				}
			})
			if allocation.Amount.Sub(consumed).IsPositive() {
				return nil, fmt.Errorf("advance attribution allocation %s exceeds remaining receivable for spend charge", allocation.Amount.String())
			}

			attributions = append(attributions, allocated...)
			unattributedAccrued[i].amount = unattributedAccrued[i].amount.Sub(allocation.Amount)

			break
		}
	}

	return attributions, nil
}

// allocateAccruedAttribution allocates a requested backfill amount across
// source-less accrued balances that have matching open advance receivable.
// Distribution across tax buckets is proportional within the selected
// occurrence; spend provenance is preserved on each attribution.
func allocateAccruedAttribution(
	calculator currencyx.Currency,
	amount alpacadecimal.Decimal,
	unattributedAccrued []unattributedAccruedBalance,
	advanceRemainingBySpendKey map[string]alpacadecimal.Decimal,
) ([]currencyx.AmountAllocation[accruedBackfillBucketKey], error) {
	items := make([]currencyx.AmountAllocationItem[accruedBackfillBucketKey], 0, len(unattributedAccrued))

	for _, balance := range unattributedAccrued {
		remaining, ok := advanceRemainingBySpendKey[balance.key.spendChargeID]
		if !ok || !remaining.IsPositive() {
			continue
		}

		if !balance.amount.IsPositive() {
			continue
		}

		items = append(items, currencyx.AmountAllocationItem[accruedBackfillBucketKey]{
			Key:    balance.key,
			Amount: balance.amount,
		})
	}

	allocations, err := currencyx.AllocateByAmount(calculator, currencyx.AmountAllocationInput[accruedBackfillBucketKey]{
		Amount:     amount,
		Items:      items,
		CompareKey: cmpx.Compare[accruedBackfillBucketKey],
	})
	if err != nil {
		return nil, fmt.Errorf("allocate accrued attribution: %w", err)
	}

	return allocations, nil
}

// totalUnattributedAccruedBalance returns accrued capacity that has matching
// open advance receivable. This caps receivable attribution so backfill does
// not translate more accrued value than exists for eligible spend provenance.
func totalUnattributedAccruedBalance(unattributedAccrued []unattributedAccruedBalance, advanceRemainingBySpendKey map[string]alpacadecimal.Decimal) alpacadecimal.Decimal {
	bySpend := make(map[string]alpacadecimal.Decimal)

	for _, balance := range unattributedAccrued {
		if balance.amount.IsPositive() {
			bySpend[balance.key.spendChargeID] = bySpend[balance.key.spendChargeID].Add(balance.amount)
		}
	}

	total := alpacadecimal.Zero

	for spend, accrued := range bySpend {
		total = total.Add(legacylineage.MinDecimal(accrued, advanceRemainingBySpendKey[spend]))
	}

	return total
}

// Multiple collection occurrences can share posting routes. Coalesce their
// attributions after FIFO selection, keeping lineage allocations occurrence-specific.
func mergeAdvanceAttributions(attributions []advanceAttribution) []advanceAttribution {
	out := make([]advanceAttribution, 0, len(attributions))

	for _, attribution := range attributions {
		idx := slices.IndexFunc(out, attribution.canMergeInto)
		if idx == -1 {
			out = append(out, attribution)
			continue
		}

		out[idx].advanceAmount = out[idx].advanceAmount.Add(attribution.advanceAmount)
		out[idx].accruedAmount = out[idx].accruedAmount.Add(attribution.accruedAmount)
	}

	return out
}
