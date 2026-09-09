package recognizer

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

type accruedKey struct {
	subAccountID   string
	sourceChargeID string
	spendChargeID  string
}

func entryAccruedKey(entry ledger.EntryInput) accruedKey {
	return accruedKey{entry.PostingAddress().SubAccountID(), lo.FromPtr(entry.SourceChargeID()), lo.FromPtr(entry.SpendChargeID())}
}

type recognitionAllocation struct {
	segment legacylineage.Segment
	amount  alpacadecimal.Decimal
}

// planRecognition intersects each segment's original backing with live accrued
// balances. Both the postings and segment transitions use this one selection;
// a customer-wide total cannot transfer recognition between unrelated sources.
func (s *service) planRecognition(ctx context.Context, in RecognizeEarningsInput, eligible []lineageEligible) ([]transactions.PostingAmount, []recognitionAllocation, error) {
	accounts, err := s.deps.AccountService.GetCustomerAccounts(ctx, in.CustomerID)
	if err != nil {
		return nil, nil, err
	}
	accountID := accounts.AccruedAccount.ID().ID
	buckets, err := s.deps.BalanceQuerier.GetBalanceBuckets(ctx, ledger.BalanceBucketQuery{
		Namespace: in.CustomerID.Namespace,
		Filters:   ledger.Filters{CollectionOriginID: mo.Some[*string](nil), AccountID: &accountID, Route: ledger.RouteFilter{Currency: in.Currency.Reference()}},
		GroupBy:   []string{ledger.BalanceBucketGroupBySourceChargeID, ledger.BalanceBucketGroupBySpendChargeID},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("load accrued balances: %w", err)
	}
	available := make(map[accruedKey]alpacadecimal.Decimal)
	for _, bucket := range buckets {
		key := accruedKey{bucket.Address.SubAccountID(), lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySourceChargeID]), lo.FromPtr(bucket.GroupByValues[ledger.BalanceBucketGroupBySpendChargeID])}
		available[key] = bucket.SettledAmount
	}

	groups := make(map[string]ledger.TransactionGroup)
	var sources []transactions.PostingAmount
	sourceIndexes := make(map[accruedKey]int)
	var allocations []recognitionAllocation
	for _, e := range eligible {
		original, err := s.recognitionGroup(ctx, in.CustomerID.Namespace, e.lineage.OriginalTransactionGroupID, groups)
		if err != nil {
			return nil, nil, err
		}
		collected, err := allocationAccruedSources(original, e.lineage.OriginalAllocationSortHint)
		if err != nil {
			return nil, nil, err
		}
		for _, segment := range e.segments {
			backing := collected
			if segment.State == creditrealization.LineageSegmentStateAdvanceBackfilled {
				group, err := s.recognitionGroup(ctx, in.CustomerID.Namespace, lo.FromPtr(segment.BackingTransactionGroupID), groups)
				if err != nil {
					return nil, nil, err
				}
				backing = backfilledAccruedSources(group, collected)
			}
			remaining := segment.Amount
			for _, source := range backing {
				entry := source.entry
				key := entryAccruedKey(entry)
				if entry.PostingAddress().Route().Route().CostBasis == nil || key.sourceChargeID == "" || key.spendChargeID == "" || key.sourceChargeID == key.spendChargeID {
					continue
				}
				take := minDecimal(remaining, minDecimal(source.amount, available[key]))
				if !take.IsPositive() {
					continue
				}
				if index, ok := sourceIndexes[key]; ok {
					sources[index].Amount = sources[index].Amount.Add(take)
				} else {
					sourceIndexes[key] = len(sources)
					sources = append(sources, transactions.PostingAmount{
						Address: entry.PostingAddress(), Amount: take,
						Identity: ledger.EntryIdentityParts{SourceChargeID: entry.SourceChargeID(), SpendChargeID: entry.SpendChargeID()},
					})
				}
				available[key] = available[key].Sub(take)
				remaining = remaining.Sub(take)
			}
			if amount := segment.Amount.Sub(remaining); amount.IsPositive() {
				allocations = append(allocations, recognitionAllocation{segment: segment, amount: amount})
			}
		}
	}
	return sources, allocations, nil
}

func (s *service) recognitionGroup(ctx context.Context, namespace, id string, groups map[string]ledger.TransactionGroup) (ledger.TransactionGroup, error) {
	if id == "" {
		return nil, fmt.Errorf("recognition lineage is missing its allocation or backing transaction group")
	}
	if group, ok := groups[id]; ok {
		return group, nil
	}
	group, err := s.ledger.GetTransactionGroup(ctx, models.NamespacedID{Namespace: namespace, ID: id})
	if err != nil {
		return nil, fmt.Errorf("load recognition source group: %w", err)
	}
	groups[id] = group
	return group, nil
}

type accruedSource struct {
	entry  ledger.Entry
	amount alpacadecimal.Decimal
}

// SortHint has the same collection contract used by correction: one allocation
// per negative FBO subaccount in transaction/entry order. A group can contain
// both promotional and paid collections, so group membership alone is insufficient.
func allocationAccruedSources(group ledger.TransactionGroup, sortHint int) ([]accruedSource, error) {
	index := 0
	for _, tx := range group.Transactions() {
		direction, err := ledger.TransactionDirectionFromAnnotations(tx.Annotations())
		if err != nil {
			return nil, err
		}
		if direction != ledger.TransactionDirectionForward {
			continue
		}
		var subAccounts []string
		bySubAccount := make(map[string][]ledger.Entry)
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerFBO || !entry.Amount().IsNegative() {
				continue
			}
			id := entry.PostingAddress().SubAccountID()
			if _, ok := bySubAccount[id]; !ok {
				subAccounts = append(subAccounts, id)
			}
			bySubAccount[id] = append(bySubAccount[id], entry)
		}
		for _, id := range subAccounts {
			if index != sortHint {
				index++
				continue
			}
			var out []accruedSource
			for _, fbo := range bySubAccount[id] {
				remaining := fbo.Amount().Abs()
				for _, accrued := range tx.Entries() {
					if accrued.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !accrued.Amount().IsPositive() {
						continue
					}
					if lo.FromPtr(fbo.SourceChargeID()) != lo.FromPtr(accrued.SourceChargeID()) || lo.FromPtr(fbo.SpendChargeID()) != lo.FromPtr(accrued.SpendChargeID()) || !sameFundingRoute(fbo.PostingAddress().Route().Route(), accrued.PostingAddress().Route().Route()) {
						continue
					}
					amount := minDecimal(remaining, accrued.Amount())
					if amount.IsPositive() {
						out = append(out, accruedSource{accrued, amount})
						remaining = remaining.Sub(amount)
					}
				}
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("allocation sort hint %d out of range for recognition source group", sortHint)
}

func sameFundingRoute(left, right ledger.Route) bool {
	if !left.Currency.Equal(right.Currency) || lo.FromPtr(left.CostBasisCurrency) != lo.FromPtr(right.CostBasisCurrency) {
		return false
	}
	if left.CostBasis == nil || right.CostBasis == nil {
		return left.CostBasis == nil && right.CostBasis == nil
	}
	return left.CostBasis.Equal(*right.CostBasis)
}

// A purchase can fund several spends and tax routes. Only its positive accrued
// translation for this original allocation backs the segment.
func backfilledAccruedSources(group ledger.TransactionGroup, original []accruedSource) []accruedSource {
	var out []accruedSource
	for _, tx := range group.Transactions() {
		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsPositive() {
				continue
			}
			route := entry.PostingAddress().Route().Route()
			for _, source := range original {
				originalRoute := source.entry.PostingAddress().Route().Route()
				if lo.FromPtr(entry.SpendChargeID()) == lo.FromPtr(source.entry.SpendChargeID()) && route.Currency.Equal(originalRoute.Currency) && lo.FromPtr(route.TaxCode) == lo.FromPtr(originalRoute.TaxCode) && lo.FromPtr(route.TaxBehavior) == lo.FromPtr(originalRoute.TaxBehavior) {
					out = append(out, accruedSource{entry, entry.Amount()})
					break
				}
			}
		}
	}
	return out
}
