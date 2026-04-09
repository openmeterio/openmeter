package lineage

import (
	"errors"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

func SortCorrectionPersistSegments(segments []Segment) []Segment {
	sorted := append([]Segment(nil), segments...)
	sort.SliceStable(sorted, func(i, j int) bool {
		precedence := func(state creditrealization.LineageSegmentState) int {
			switch state {
			case creditrealization.LineageSegmentStateAdvanceBackfilled:
				return 0
			case creditrealization.LineageSegmentStateAdvanceUncovered:
				return 1
			case creditrealization.LineageSegmentStateRealCredit:
				return 2
			default:
				return 3
			}
		}

		return precedence(sorted[i].State) < precedence(sorted[j].State)
	})

	return sorted
}

func MinDecimal(a, b alpacadecimal.Decimal) alpacadecimal.Decimal {
	if a.GreaterThan(b) {
		return b
	}

	return a
}

func (s Segment) Validate() error {
	var errs []error

	if !s.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if err := s.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	if s.State == creditrealization.LineageSegmentStateAdvanceBackfilled && (s.BackingTransactionGroupID == nil || *s.BackingTransactionGroupID == "") {
		errs = append(errs, errors.New("backing transaction group id is required for advance_backfilled"))
	}

	return errors.Join(errs...)
}
