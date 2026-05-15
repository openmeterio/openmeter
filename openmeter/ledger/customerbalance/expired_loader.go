package customerbalance

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type expiredCreditTransactionLoader struct {
	service *service
}

func newExpiredCreditTransactionLoader(s *service) creditTransactionLoader {
	return &expiredCreditTransactionLoader{service: s}
}

func (l *expiredCreditTransactionLoader) Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error) {
	records, err := l.service.Breakage.ListExpiredRecords(ctx, breakage.ListExpiredRecordsInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       input.AsOf,
	})
	if err != nil {
		return creditTransactionLoaderResult{}, fmt.Errorf("list expired breakage records: %w", err)
	}

	groups := make(map[expiredBreakageGroupKey]*expiredBreakageGroup)
	for _, record := range records {
		key := expiredBreakageGroupKey{
			expiresAt: record.ExpiresAt,
			currency:  record.Currency,
		}

		group := groups[key]
		if group == nil {
			group = &expiredBreakageGroup{
				expiresAt: record.ExpiresAt,
				currency:  record.Currency,
			}
			groups[key] = group
		}

		switch record.Kind {
		case ledger.BreakageKindPlan, ledger.BreakageKindReopen:
			group.amount = group.amount.Add(record.Amount)
		case ledger.BreakageKindRelease:
			group.amount = group.amount.Sub(record.Amount)
		default:
			return creditTransactionLoaderResult{}, fmt.Errorf("unexpected breakage kind %q", record.Kind)
		}

		group.transactionRefs = append(group.transactionRefs, breakageTransactionRef{
			groupID:       record.BreakageTransactionGroupID,
			transactionID: record.BreakageTransactionID,
		})
	}

	items := make([]CreditTransaction, 0, len(groups))
	cursorCache := map[breakageTransactionRef]ledger.TransactionCursor{}

	for _, group := range groups {
		if group.amount.IsZero() {
			continue
		}
		if group.amount.IsNegative() {
			return creditTransactionLoaderResult{}, fmt.Errorf("expired breakage amount is negative for %s %s", group.expiresAt, group.currency)
		}

		cursor, err := l.newestBreakageTransactionCursor(ctx, input.CustomerID.Namespace, group.transactionRefs, cursorCache)
		if err != nil {
			return creditTransactionLoaderResult{}, err
		}

		item := CreditTransaction{
			ID:        cursor.ID,
			CreatedAt: cursor.CreatedAt,
			BookedAt:  group.expiresAt,
			Type:      CreditTransactionTypeExpired,
			Currency:  group.currency,
			Amount:    group.amount.Neg(),
			Name:      "Expired credits",
			Annotations: models.Annotations{
				ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
			},
		}

		if !creditTransactionMatchesCursorWindow(item, input.After, input.Before) {
			continue
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b CreditTransaction) int {
		return -compareCreditTransactionsByCursor(a, b)
	})

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}

	return creditTransactionLoaderResult{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func (l *expiredCreditTransactionLoader) newestBreakageTransactionCursor(
	ctx context.Context,
	namespace string,
	refs []breakageTransactionRef,
	cache map[breakageTransactionRef]ledger.TransactionCursor,
) (ledger.TransactionCursor, error) {
	var newest *ledger.TransactionCursor

	for _, ref := range refs {
		cursor, ok := cache[ref]
		if !ok {
			group, err := l.service.Ledger.GetTransactionGroup(ctx, models.NamespacedID{
				Namespace: namespace,
				ID:        ref.groupID,
			})
			if err != nil {
				return ledger.TransactionCursor{}, fmt.Errorf("get breakage transaction group %s: %w", ref.groupID, err)
			}

			found := false
			for _, tx := range group.Transactions() {
				if tx.ID().ID != ref.transactionID {
					continue
				}

				cursor = tx.Cursor()
				found = true
				break
			}
			if !found {
				return ledger.TransactionCursor{}, fmt.Errorf("breakage transaction %s not found in group %s", ref.transactionID, ref.groupID)
			}

			cache[ref] = cursor
		}

		if newest == nil || cursor.Compare(*newest) > 0 {
			cursorCopy := cursor
			newest = &cursorCopy
		}
	}

	if newest == nil {
		return ledger.TransactionCursor{}, fmt.Errorf("expired breakage group has no transaction references")
	}

	return *newest, nil
}

func creditTransactionMatchesCursorWindow(item CreditTransaction, after, before *ledger.TransactionCursor) bool {
	cursor := creditTransactionCursor(item)

	if after != nil && cursor.Compare(*after) >= 0 {
		return false
	}

	if before != nil && cursor.Compare(*before) <= 0 {
		return false
	}

	return true
}

type expiredBreakageGroupKey struct {
	expiresAt time.Time
	currency  currencyx.Code
}

type expiredBreakageGroup struct {
	expiresAt       time.Time
	currency        currencyx.Code
	amount          alpacadecimal.Decimal
	transactionRefs []breakageTransactionRef
}

type breakageTransactionRef struct {
	groupID       string
	transactionID string
}
