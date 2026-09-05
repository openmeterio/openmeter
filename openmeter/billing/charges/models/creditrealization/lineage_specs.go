package creditrealization

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type InitialLineageSpec struct {
	LineageID         string
	RootRealizationID string
	OriginKind        LineageOriginKind
	InitialState      LineageSegmentState
	Amount            alpacadecimal.Decimal
	AdvanceFeatures   []string
}

func InitialLineageSpecs(realizations Realizations) ([]InitialLineageSpec, error) {
	out := make([]InitialLineageSpec, 0, len(realizations))

	for _, realization := range realizations {
		if realization.Annotations[ledger.AnnotationOriginTracked] == true {
			continue
		}
		if realization.Type != TypeAllocation {
			continue
		}

		originKind, err := LineageOriginKindFromAnnotations(realization.Annotations)
		if err != nil {
			continue
		}

		initialState := InitialLineageSegmentState(originKind)
		if err := initialState.Validate(); err != nil {
			return nil, fmt.Errorf("realization %s initial lineage state: %w", realization.ID, err)
		}

		out = append(out, InitialLineageSpec{
			LineageID:         ulid.Make().String(),
			RootRealizationID: realization.ID,
			OriginKind:        originKind,
			InitialState:      initialState,
			Amount:            realization.Amount,
		})
	}

	return out, nil
}

// LegacyLineageRealizations selects histories whose amounts still depend on the
// pre-origin compatibility system. Tracked allocations and their corrections
// are represented entirely by ledger entries.
func (r Realizations) LegacyLineageRealizations() Realizations {
	var out Realizations
	for _, realization := range r {
		if realization.Annotations[ledger.AnnotationOriginTracked] != true {
			out = append(out, realization)
		}
	}
	return out
}
