package service

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/advance"
	"github.com/openmeterio/openmeter/openmeter/ledger/crediteligibility"
	"github.com/openmeterio/openmeter/pkg/cmpx"
)

// selectBackfill matches receivable and accrued by spend and collection origin.
// Legacy nil-spend balances remain separated by their original feature routes.
func (s *service) selectBackfill(ctx context.Context, input advance.BackfillInput) (backfillSelection, error) {
	currencyReference := input.Currency.Reference()

	customerAccounts, err := s.accountResolver.GetCustomerAccounts(ctx, input.CustomerID)
	if err != nil {
		return backfillSelection{}, fmt.Errorf("get customer accounts: %w", err)
	}

	advanceReceivables, err := s.advanceReceivableBalances(ctx, customerAccounts.ReceivableAccount.ID(), currencyReference)
	if err != nil {
		return backfillSelection{}, fmt.Errorf("list advance receivable balances: %w", err)
	}

	slices.SortStableFunc(advanceReceivables, cmpx.Compare[advanceReceivableBalance])

	unattributedAccrued, err := s.unattributedAccruedBalances(ctx, customerAccounts.AccruedAccount, currencyReference)
	if err != nil {
		return backfillSelection{}, err
	}

	receivableBuckets := newAdvanceReceivableBuckets(advanceReceivables, input.Filters)

	plan := backfillSelection{}
	remaining := input.Amount
	candidates := advanceBackfillCandidates(input.LegacyLineages, unattributedAccrued, input.Filters)

	for _, candidate := range candidates {
		if !remaining.IsPositive() {
			break
		}

		var spendKey string

		var accruedBuckets []unattributedAccruedBalance

		var selections []legacylineage.AdvanceBackfillAllocation

		if candidate.legacy != nil {
			root := *candidate.legacy
			receivableBuckets.requiredFilters = crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: root.AdvanceFeatures}

			spendKey, accruedBuckets, err = s.accruedBucketsForAdvance(ctx, input.CustomerID.Namespace, root, unattributedAccrued)
			if err != nil {
				return backfillSelection{}, err
			}

			for _, segment := range root.Segments {
				selections = append(selections, legacylineage.AdvanceBackfillAllocation{
					SegmentID: segment.ID,
					Amount:    segment.Amount,
				})
			}
		} else {
			spendKey = candidate.key.spendChargeID

			for _, balance := range unattributedAccrued {
				if balance.key == candidate.key {
					accruedBuckets = append(accruedBuckets, balance)
				}
			}

			balances := receivableBuckets.bySpendChargeID[spendKey]
			if len(balances) == 0 {
				continue
			}

			receivableBuckets.requiredFilters = balances[0].address.Route().Route().Filters
			selections = []legacylineage.AdvanceBackfillAllocation{{Amount: remaining}}
		}

		for _, selection := range selections {
			if !remaining.IsPositive() {
				break
			}

			receivableCapacity := receivableBuckets.availableForSpend(spendKey)
			capacity := totalUnattributedAccruedBalance(accruedBuckets, map[string]alpacadecimal.Decimal{spendKey: receivableCapacity})
			covered := legacylineage.MinDecimal(selection.Amount, legacylineage.MinDecimal(remaining, capacity))
			if !covered.IsPositive() {
				continue
			}

			allocations, err := allocateAccruedAttribution(input.Currency, covered, accruedBuckets, map[string]alpacadecimal.Decimal{spendKey: covered})
			if err != nil {
				return backfillSelection{}, err
			}

			attributions, err := allocateAccruedBackedAdvanceAttributions(allocations, unattributedAccrued, &receivableBuckets)
			if err != nil {
				return backfillSelection{}, err
			}

			// Keep the occurrence capacity in step with the shared balances before
			// considering another legacy segment of the same root.
			for i := range accruedBuckets {
				for _, allocation := range allocations {
					if accruedBuckets[i].key == allocation.Key {
						accruedBuckets[i].amount = accruedBuckets[i].amount.Sub(allocation.Amount)
					}
				}
			}

			plan.attributions = append(plan.attributions, attributions...)
			if candidate.legacy != nil {
				plan.allocations = append(plan.allocations, legacylineage.AdvanceBackfillAllocation{
					SegmentID: selection.SegmentID,
					Amount:    covered,
				})
			}

			remaining = remaining.Sub(covered)
		}
	}

	// Remaining receivable can be attributed even without matching accrued.
	// Only the accrued-backed amounts above become lineage backfill allocations.
	plan.attributions = append(plan.attributions, receivableBuckets.attributeRemaining(remaining)...)

	return plan, nil
}

type backfillSelection struct {
	attributions []advanceAttribution
	allocations  []legacylineage.AdvanceBackfillAllocation
}

// advanceBackfillCandidate is transient selection data. Origin capacity comes
// from journal sums; only legacy candidates carry persisted segment state.
type advanceBackfillCandidate struct {
	recordedAt time.Time
	id         string
	key        accruedBackfillBucketKey
	legacy     *legacylineage.Lineage
}

func advanceBackfillCandidates(roots []legacylineage.Lineage, balances []unattributedAccruedBalance, filters crediteligibility.Filters) []advanceBackfillCandidate {
	var candidates []advanceBackfillCandidate

	for _, root := range sortedAdvanceBackfillLineages(roots) {
		if !filters.Matches(ledger.Route{Filters: crediteligibility.Filters{Version: crediteligibility.FiltersVersion1, Features: root.AdvanceFeatures}}) {
			continue
		}
		candidates = append(candidates, advanceBackfillCandidate{
			recordedAt: root.CreatedAt,
			id:         root.ID,
			legacy:     &root,
		})
	}

	for _, balance := range balances {
		if balance.collectionOriginID == nil || !balance.amount.IsPositive() {
			continue
		}

		candidates = append(candidates, advanceBackfillCandidate{
			recordedAt: balance.oldestMatchingEntryCreatedAt,
			id:         *balance.collectionOriginID,
			key:        balance.key,
		})
	}

	slices.SortFunc(candidates, func(a, b advanceBackfillCandidate) int {
		return cmp.Or(a.recordedAt.Compare(b.recordedAt), cmp.Compare(a.id, b.id), cmpx.Compare(a.key, b.key))
	})

	return candidates
}
