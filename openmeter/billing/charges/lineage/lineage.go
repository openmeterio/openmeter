package lineage

import (
	"sort"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

func AttachInitialActiveLineageSegments(realizations creditrealization.Realizations) {
	for idx := range realizations {
		originKind, err := creditrealization.LineageOriginKindFromAnnotations(realizations[idx].Annotations)
		if err != nil {
			continue
		}

		initialState := creditrealization.InitialLineageSegmentState(originKind)
		if err := initialState.Validate(); err != nil {
			continue
		}

		realizations[idx].ActiveLineageSegments = []creditrealization.ActiveLineageSegment{
			{
				Amount: realizations[idx].Amount,
				State:  initialState,
			},
		}
	}
}

func SortCorrectionWritebackSegments(segments []Segment) []Segment {
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
