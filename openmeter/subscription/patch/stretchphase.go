package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

type PatchStretchPhase struct {
	PhaseKey string
	// Signed duration
	Duration isodate.Period
}

func (p PatchStretchPhase) Op() subscription.PatchOperation {
	return subscription.PatchOperationStretch
}

func (p PatchStretchPhase) Path() subscription.SpecPath {
	return subscription.NewPhasePath(p.PhaseKey)
}

func (p PatchStretchPhase) Value() isodate.Period {
	return p.Duration
}

func (p PatchStretchPhase) ValueAsAny() any {
	return p.Duration
}

func (p PatchStretchPhase) Validate() error {
	if err := p.Path().Validate(); err != nil {
		return err
	}

	if err := p.Op().Validate(); err != nil {
		return err
	}

	if p.Duration.IsZero() {
		return fmt.Errorf("duration cannot be zero")
	}

	return nil
}

var _ subscription.ValuePatch[isodate.Period] = PatchStretchPhase{}

func (p PatchStretchPhase) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, ok := spec.Phases[p.PhaseKey]
	if !ok {
		return fmt.Errorf("phase %s not found", p.PhaseKey)
	}

	sortedPhases := spec.GetSortedPhases()

	// Checks we need:
	pST, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
	// 2. You cannot extend past phases, only current or future ones
	current, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
	if exists {
		cPST, _ := current.StartAfter.AddTo(spec.ActiveFrom)

		if pST.Before(cPST) {
			return &subscription.PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	} else {
		// If current phase doesn't exist then all phases are either in the past or in the future
		// If they're all in the past then the by checking any we can see if it should fail or not
		if pST.Before(actx.CurrentTime) {
			return &subscription.PatchForbiddenError{Msg: "cannot extend past phase"}
		}
	}

	if len(sortedPhases) < 2 {
		return &subscription.PatchConflictError{Msg: "cannot stretch a single phase"}
	}

	reachedTargetPhase := false
	for i, thisP := range sortedPhases {
		if thisP.PhaseKey == p.PhaseKey {
			reachedTargetPhase = true
			continue
		}

		if reachedTargetPhase {
			// Adding durtions in the semantic way (using ISO8601 format)
			sa, err := thisP.StartAfter.Add(p.Duration)
			if err != nil {
				return &subscription.PatchValidationError{Msg: fmt.Sprintf("failed to extend phase %s: %s", thisP.PhaseKey, err)}
			}

			// before changing lets make sure the previous phase doesn't disappear
			if i > 0 {
				prev := sortedPhases[i-1]
				prevStart, _ := prev.StartAfter.AddTo(spec.ActiveFrom)
				newStart, _ := sa.AddTo(spec.ActiveFrom)
				if !newStart.After(prevStart) {
					return &subscription.PatchConflictError{Msg: fmt.Sprintf("phase %s would disappear due to stretching", prev.PhaseKey)}
				}
			}

			sortedPhases[i].StartAfter = sa
		}
	}

	return nil
}
