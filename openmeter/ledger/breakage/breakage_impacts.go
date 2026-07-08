package breakage

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

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

	if err := ValidateExpiredRouteFilter(i.Route); err != nil {
		errs = append(errs, fmt.Errorf("route: %w", err))
	}

	if i.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
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
		Route:      input.Route,
	})
	if err != nil {
		return ListExpiredBreakageImpactsResult{}, fmt.Errorf("list expired breakage records: %w", err)
	}

	groups := make(map[expiredBreakageImpactGroupKey]*expiredBreakageImpactGroup)
	for _, record := range records {
		key := expiredBreakageImpactGroupKey{
			expiresAt:      record.ExpiresAt,
			currency:       record.Currency,
			sourceChargeID: lo.FromPtr(record.SourceChargeID),
		}

		group := groups[key]
		if group == nil {
			group = &expiredBreakageImpactGroup{
				expiresAt:      record.ExpiresAt,
				currency:       record.Currency,
				sourceChargeID: record.SourceChargeID,
			}
			groups[key] = group
		}

		switch record.Kind {
		case ledger.BreakageKindPlan, ledger.BreakageKindReopen:
			group.amount = group.amount.Add(record.Amount)
			if record.Kind == ledger.BreakageKindPlan && (group.cursorID.ID == "" || record.ID.ID < group.cursorID.ID) {
				group.cursorID = record.ID
			}
		case ledger.BreakageKindRelease:
			group.amount = group.amount.Sub(record.Amount)
		default:
			return ListExpiredBreakageImpactsResult{}, fmt.Errorf("unexpected breakage kind %q", record.Kind)
		}
	}
	if len(groups) == 0 {
		return ListExpiredBreakageImpactsResult{
			Items: []BreakageImpact{},
		}, nil
	}

	items := make([]BreakageImpact, 0, len(groups))

	for _, group := range groups {
		if group.amount.IsZero() {
			continue
		}
		if group.amount.IsNegative() {
			return ListExpiredBreakageImpactsResult{}, fmt.Errorf("expired breakage amount is negative for %s %s", group.expiresAt, group.currency)
		}
		if group.cursorID.ID == "" {
			return ListExpiredBreakageImpactsResult{}, fmt.Errorf("expired breakage impact has no plan record for %s %s", group.expiresAt, group.currency)
		}

		annotations := models.Annotations{
			ledger.AnnotationCollectionType: ledger.CollectionTypeBreakage,
		}
		if group.sourceChargeID != nil {
			annotations[ledger.AnnotationChargeID] = *group.sourceChargeID
		}

		item := BreakageImpact{
			ID:          group.cursorID,
			CreatedAt:   group.expiresAt,
			BookedAt:    group.expiresAt,
			CustomerID:  input.CustomerID,
			Currency:    group.currency,
			Amount:      group.amount.Neg(),
			SourceKind:  SourceKindCreditPurchase,
			Annotations: annotations,
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

func ValidateExpiredRouteFilter(route ledger.RouteFilter) error {
	var errs []error

	if route.Features.IsPresent() && route.MatchFeature != "" {
		errs = append(errs, errors.New("features and match feature filters cannot be combined"))
	}

	if route.Features.IsPresent() {
		features, _ := route.Features.Get()
		if err := ledger.ValidateFeatures(features); err != nil {
			errs = append(errs, fmt.Errorf("features: %w", err))
		}
	}

	if route.MatchFeature != "" {
		if err := ledger.ValidateFeatures([]string{route.MatchFeature}); err != nil {
			errs = append(errs, fmt.Errorf("match feature: %w", err))
		}
	}

	return errors.Join(errs...)
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
	expiresAt      time.Time
	currency       currencyx.Code
	sourceChargeID string
}

type expiredBreakageImpactGroup struct {
	expiresAt      time.Time
	currency       currencyx.Code
	sourceChargeID *string
	amount         alpacadecimal.Decimal
	cursorID       models.NamespacedID
}
