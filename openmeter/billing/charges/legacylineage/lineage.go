package legacylineage

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

func MinDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}

func FilterAdvanceLineagesForBackfill(lineages []Lineage, featureFilters []string) []Lineage {
	return lo.Filter(lineages, func(entry Lineage, _ int) bool {
		return FeatureFiltersMatchAdvance(featureFilters, entry.AdvanceFeatures)
	})
}

func FeatureFiltersMatchAdvance(featureFilters []string, advanceFeatures []string) bool {
	if len(featureFilters) == 0 {
		return true
	}

	if len(advanceFeatures) == 0 {
		return false
	}

	for _, feature := range advanceFeatures {
		if lo.Contains(featureFilters, feature) {
			return true
		}
	}

	return false
}

func (s Segment) Validate() error {
	var errs []error

	if !s.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if err := s.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	switch s.State {
	case creditrealization.LineageSegmentStateAdvanceBackfilled:
		if s.BackingTransactionGroupID == nil || *s.BackingTransactionGroupID == "" {
			errs = append(errs, errors.New("backing transaction group id is required for advance_backfilled"))
		}
	case creditrealization.LineageSegmentStateEarningsRecognized:
		if s.BackingTransactionGroupID == nil || *s.BackingTransactionGroupID == "" {
			errs = append(errs, errors.New("backing transaction group id is required for earnings_recognized"))
		}
		switch {
		case s.SourceState == nil:
			errs = append(errs, errors.New("source state is required for earnings_recognized"))
		case *s.SourceState == creditrealization.LineageSegmentStateEarningsRecognized:
			errs = append(errs, errors.New("source state cannot be earnings_recognized"))
		case *s.SourceState == creditrealization.LineageSegmentStateAdvanceBackfilled:
			if s.SourceBackingTransactionGroupID == nil || *s.SourceBackingTransactionGroupID == "" {
				errs = append(errs, errors.New("source backing transaction group id is required when source state is advance_backfilled"))
			}
		}
	}

	return errors.Join(errs...)
}
