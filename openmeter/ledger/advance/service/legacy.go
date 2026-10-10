package service

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/legacylineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) accruedBucketsForAdvance(ctx context.Context, namespace string, root legacylineage.Lineage, balances []unattributedAccruedBalance) (string, []unattributedAccruedBalance, error) {
	if root.OriginalTransactionGroupID == "" {
		return "", nil, fmt.Errorf("advance lineage %s is missing its original transaction group", root.ID)
	}

	group, err := s.ledger.GetTransactionGroup(ctx, models.NamespacedID{
		Namespace: namespace,
		ID:        root.OriginalTransactionGroupID,
	})
	if err != nil {
		return "", nil, err
	}

	key, err := originalAdvanceAccruedBucket(group)
	if err != nil {
		return "", nil, err
	}

	var matched []unattributedAccruedBalance

	for _, balance := range balances {
		if balance.key == key {
			matched = append(matched, balance)
		}
	}

	return key.spendChargeID, matched, nil
}

// sortedAdvanceBackfillLineages makes FIFO independent of caller/query order.
// Collection time orders occurrences; replacement times only order segments
// within an occurrence. Copies preserve the caller's slices and omit history
// that cannot consume purchase value before any journal is loaded.
func sortedAdvanceBackfillLineages(roots []legacylineage.Lineage) []legacylineage.Lineage {
	candidates := make([]legacylineage.Lineage, 0, len(roots))

	for _, root := range roots {
		if root.OriginKind != creditrealization.LineageOriginKindAdvance {
			continue
		}

		root.Segments = lo.Filter(root.Segments, func(segment legacylineage.Segment, _ int) bool {
			return segment.State == creditrealization.LineageSegmentStateAdvanceUncovered
		})
		if len(root.Segments) == 0 {
			continue
		}

		slices.SortFunc(root.Segments, func(a, b legacylineage.Segment) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})

		candidates = append(candidates, root)
	}

	slices.SortFunc(candidates, func(a, b legacylineage.Lineage) int {
		return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
	})

	return candidates
}

// originalAdvanceAccruedBucket reads the original tax route and spend provenance,
// including nil-spend legacy collections, rather than inferring them from charge metadata.
func originalAdvanceAccruedBucket(group ledger.TransactionGroup) (accruedBackfillBucketKey, error) {
	for _, tx := range group.Transactions() {
		code, err := ledger.TransactionTemplateCodeFromAnnotations(tx.Annotations())
		if err != nil {
			return accruedBackfillBucketKey{}, err
		}

		if code != transactions.TemplateCode(transactions.TransferCustomerFBOAdvanceToAccruedTemplate{}) {
			continue
		}

		for _, entry := range tx.Entries() {
			if entry.PostingAddress().AccountType() != ledger.AccountTypeCustomerAccrued || !entry.Amount().IsPositive() {
				continue
			}

			return accruedBackfillBucketKey{
				spendChargeID:   lo.FromPtrOr(entry.Provenance().SpendChargeID, "null"),
				taxDimensionKey: taxDimensionRouteKey(entry.PostingAddress().Route().Route()),
			}, nil
		}
	}

	return accruedBackfillBucketKey{}, fmt.Errorf("original advance collection missing from group %s", group.ID().ID)
}
