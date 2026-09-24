package customerbalance

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
)

type refundedCreditTransactionLoader struct {
	service *service
}

func newRefundedCreditTransactionLoader(s *service) creditTransactionLoader {
	return &refundedCreditTransactionLoader{service: s}
}

func (l *refundedCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	return l.service.loadCandidateCreditTransactions(ctx, input, map[string]string{
		ledger.AnnotationTransactionDirection: string(ledger.TransactionDirectionCorrection),
	}, func(ctx context.Context, candidates []ledger.Transaction, limit int) ([]CreditTransaction, error) {
		items := make([]CreditTransaction, 0, min(len(candidates), limit))
		refundsByGroupID := make(map[string][]CreditTransaction)

		for _, candidate := range candidates {
			if !isRefundTransaction(candidate) {
				continue
			}

			groupID := candidate.GroupID()
			refunds, ok := refundsByGroupID[groupID.ID]
			if !ok {
				group, err := l.service.Ledger.GetTransactionGroup(ctx, groupID)
				if err != nil {
					return nil, fmt.Errorf("get correction transaction group %s: %w", groupID.ID, err)
				}

				refunds, err = refundedCreditTransactionsFromGroup(group, input)
				if err != nil {
					return nil, fmt.Errorf("resolve correction transaction group %s: %w", groupID.ID, err)
				}
				refundsByGroupID[groupID.ID] = refunds
			}

			// Each refund is emitted once, at its latest contributing transaction.
			candidateCursor := candidate.Cursor()
			for _, refund := range refunds {
				if refund.balanceCursor.Compare(candidateCursor) == 0 &&
					creditTransactionMatchesCursorWindow(refund, input.After, input.Before) {
					items = append(items, refund)
					break
				}
			}

			if len(items) == limit {
				break
			}
		}

		return items, nil
	})
}

// isRefundTransaction reports whether a correction returns consumed credit.
// Credit voids are also ledger corrections, but they forfeit funded credit and
// are listed as voided.
func isRefundTransaction(tx ledger.Transaction) bool {
	annotations := tx.Annotations()
	if annotations[ledger.AnnotationTransactionDirection] != string(ledger.TransactionDirectionCorrection) ||
		annotations[ledger.AnnotationCollectionType] == ledger.CollectionTypeBreakage ||
		annotations[ledger.AnnotationCustomerBalanceVisibility] == ledger.CustomerBalanceVisibilityInternal {
		return false
	}

	_, isVoid := annotations[creditvoid.AnnotationCreditVoidRecordID]
	return !isVoid
}

// refundedCreditTransactionsFromGroup nets a correction group's customer
// balance impact per booked time and currency. One usage correction can combine
// collection reversals, advance cancellation, and backfill unwinding whose
// individual impacts offset each other; only their sum is refunded credit.
func refundedCreditTransactionsFromGroup(group ledger.TransactionGroup, input creditTransactionLoaderInput) ([]CreditTransaction, error) {
	type refundKey struct {
		bookedAt time.Time
		currency string
	}

	refundsByKey := make(map[refundKey]CreditTransaction)
	for _, tx := range group.Transactions() {
		if !isRefundTransaction(tx) || tx.BookedAt().After(input.AsOf) {
			continue
		}

		impact, currency, err := creditTransactionBalanceImpact(tx, GetBalanceServiceInput{
			FeatureFilter: input.FeatureFilter,
		})
		if err != nil {
			return nil, err
		}
		if impact.IsZero() {
			continue
		}
		if input.Currency != nil && currency.Code != *input.Currency {
			continue
		}

		key := refundKey{
			bookedAt: tx.BookedAt().UTC(),
			currency: currency.IdentityKey(),
		}
		cursor := tx.Cursor()

		refund, ok := refundsByKey[key]
		if !ok {
			refund = CreditTransaction{
				Type:             CreditTransactionTypeRefunded,
				Currency:         currency.GetCode(),
				CustomCurrencyID: currency.CustomCurrencyID,
				Amount:           alpacadecimal.Zero,
			}
		}

		refund.Amount = refund.Amount.Add(impact)
		if refund.balanceCursor == nil || refund.balanceCursor.Compare(cursor) < 0 {
			refund.ID = tx.ID()
			refund.CreatedAt = cursor.CreatedAt
			refund.BookedAt = cursor.BookedAt
			refund.Annotations = tx.Annotations()
			refund.balanceCursor = &cursor
		}
		refundsByKey[key] = refund
	}

	refunds := make([]CreditTransaction, 0, len(refundsByKey))
	for _, refund := range refundsByKey {
		if refund.Amount.IsZero() {
			continue
		}

		impact := refund.Amount
		refund.balanceImpact = &impact
		refunds = append(refunds, refund)
	}
	slices.SortFunc(refunds, func(a, b CreditTransaction) int {
		return b.balanceCursor.Compare(*a.balanceCursor)
	})

	return refunds, nil
}
