package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type PatchStretchPhase struct {
	PhaseKey string
	// Signed duration
	Duration datex.Period
}

func (p PatchStretchPhase) Op() subscription.PatchOperation {
	return subscription.PatchOperationStretch
}

func (p PatchStretchPhase) Path() subscription.PatchPath {
	return subscription.NewPhasePath(p.PhaseKey)
}

func (p PatchStretchPhase) Value() datex.Period {
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

var _ subscription.ValuePatch[datex.Period] = PatchStretchPhase{}

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

	reachedTargetPhase := false
	for i, thisP := range sortedPhases {
		if thisP.PhaseKey == p.PhaseKey {
			reachedTargetPhase = true
		}

		if reachedTargetPhase {
			// Adding durtions in the semantic way (using ISO8601 format)
			sa, err := thisP.StartAfter.Add(p.Duration)
			if err != nil {
				return &subscription.PatchValidationError{Msg: fmt.Sprintf("failed to extend phase %s: %s", thisP.PhaseKey, err)}
			}
			sortedPhases[i].StartAfter = sa
		}
	}

	return nil
}
