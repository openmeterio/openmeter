package breakage

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (i ListExpiredBreakageImpactsInput) Validate() error {
	var errs []error

	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer id: %w", err))
	}

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	if i.AsOf.IsZero() {
		errs = append(errs, errors.New("as of is required"))
	}

	if i.After != nil {
		if err := i.After.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("after: %w", err))
		}
	}

	if i.Before != nil {
		if err := i.Before.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("before: %w", err))
		}
	}

	if i.After != nil && i.Before != nil {
		errs = append(errs, errors.New("after and before cannot be set together"))
	}

	if i.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	return errors.Join(errs...)
}

func (i ListBreakageTransactionCursorsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if len(i.TransactionID) == 0 {
		errs = append(errs, errors.New("transaction ids are required"))
	}

	for idx, id := range i.TransactionID {
		if id == "" {
			errs = append(errs, fmt.Errorf("transaction ids[%d] is required", idx))
		}
	}

	return errors.Join(errs...)
}

func (s *service) ListExpiredBreakageImpacts(ctx context.Context, input ListExpiredBreakageImpactsInput) (ListExpiredBreakageImpactsResult, error) {
	if err := input.Validate(); err != nil {
		return ListExpiredBreakageImpactsResult{}, err
	}

	records, err := s.ListExpiredRecords(ctx, ListExpiredRecordsInput{
		CustomerID: input.CustomerID,
		Currency:   input.Currency,
		AsOf:       input.AsOf,
	})
	if err != nil {
		return ListExpiredBreakageImpactsResult{}, fmt.Errorf("list expired breakage records: %w", err)
	}

	groups := make(map[expiredBreakageImpactGroupKey]*expiredBreakageImpactGroup)
	transactionIDs := make([]string, 0, len(records))
	transactionIDSeen := make(map[string]struct{}, len(records))
	for _, record := range records {
		key := expiredBreakageImpactGroupKey{
			expiresAt: record.ExpiresAt,
			currency:  record.Currency,
		}

		group := groups[key]
		if group == nil {
			group = &expiredBreakageImpactGroup{
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
			return ListExpiredBreakageImpactsResult{}, fmt.Errorf("unexpected breakage kind %q", record.Kind)
		}

		group.transactionIDs = append(group.transactionIDs, record.BreakageTransactionID)
		if _, ok := transactionIDSeen[record.BreakageTransactionID]; !ok {
			transactionIDs = append(transactionIDs, record.BreakageTransactionID)
			transactionIDSeen[record.BreakageTransactionID] = struct{}{}
		}
	}
	if len(transactionIDs) == 0 {
		return ListExpiredBreakageImpactsResult{
			Items: []BreakageImpact{},
		}, nil
	}

	transactionCursors, err := s.adapter.ListBreakageTransactionCursors(ctx, ListBreakageTransactionCursorsInput{
		Namespace:     input.CustomerID.Namespace,
		TransactionID: transactionIDs,
	})
	if err != nil {
		return ListExpiredBreakageImpactsResult{}, fmt.Errorf("list breakage transaction cursors: %w", err)
	}

	items := make([]BreakageImpact, 0, len(groups))

	for _, group := range groups {
		if group.amount.IsZero() {
			continue
		}
		if group.amount.IsNegative() {
			return ListExpiredBreakageImpactsResult{}, fmt.Errorf("expired breakage amount is negative for %s %s", group.expiresAt, group.currency)
		}

		cursor, err := newestBreakageImpactCursor(group.transactionIDs, transactionCursors)
		if err != nil {
			return ListExpiredBreakageImpactsResult{}, err
		}

		item := BreakageImpact{
			ID:         cursor.ID,
			CreatedAt:  cursor.CreatedAt,
			BookedAt:   group.expiresAt,
			CustomerID: input.CustomerID,
			Currency:   group.currency,
			Amount:     group.amount.Neg(),
			Annotations: models.Annotations{
				ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
			},
		}

		if !breakageImpactMatchesCursorWindow(item, input.After, input.Before) {
			continue
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b BreakageImpact) int {
		return -a.Cursor().Compare(b.Cursor())
	})

	hasMore := len(items) > input.Limit
	if hasMore {
		items = items[:input.Limit]
	}

	return ListExpiredBreakageImpactsResult{
		Items:   items,
		HasMore: hasMore,
	}, nil
}

func newestBreakageImpactCursor(transactionIDs []string, cursors map[string]ledger.TransactionCursor) (ledger.TransactionCursor, error) {
	var newest *ledger.TransactionCursor

	for _, transactionID := range transactionIDs {
		cursor, ok := cursors[transactionID]
		if !ok {
			return ledger.TransactionCursor{}, fmt.Errorf("breakage transaction cursor %s not found", transactionID)
		}

		if newest == nil || cursor.Compare(*newest) > 0 {
			cursorCopy := cursor
			newest = &cursorCopy
		}
	}

	if newest == nil {
		return ledger.TransactionCursor{}, errors.New("expired breakage impact has no transaction references")
	}

	return *newest, nil
}

func breakageImpactMatchesCursorWindow(item BreakageImpact, after, before *ledger.TransactionCursor) bool {
	cursor := item.Cursor()

	if after != nil && cursor.Compare(*after) >= 0 {
		return false
	}

	if before != nil && cursor.Compare(*before) <= 0 {
		return false
	}

	return true
}

type expiredBreakageImpactGroupKey struct {
	expiresAt time.Time
	currency  currencyx.Code
}

type expiredBreakageImpactGroup struct {
	expiresAt      time.Time
	currency       currencyx.Code
	amount         alpacadecimal.Decimal
	transactionIDs []string
}
