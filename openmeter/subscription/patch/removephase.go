package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

type PatchRemovePhase struct {
	PhaseKey    string
	RemoveInput subscription.RemoveSubscriptionPhaseInput
}

func (r PatchRemovePhase) Op() subscription.PatchOperation {
	return subscription.PatchOperationRemove
}

func (r PatchRemovePhase) Path() subscription.SpecPath {
	return subscription.NewPhasePath(r.PhaseKey)
}

func (r PatchRemovePhase) Value() subscription.RemoveSubscriptionPhaseInput {
	return r.RemoveInput
}

func (r PatchRemovePhase) ValueAsAny() any {
	return r.RemoveInput
}

func (r PatchRemovePhase) Validate() error {
	if err := r.Path().Validate(); err != nil {
		return err
	}

	if err := r.Op().Validate(); err != nil {
		return err
	}

	return nil
}

var _ subscription.ValuePatch[subscription.RemoveSubscriptionPhaseInput] = PatchRemovePhase{}

func (r PatchRemovePhase) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, exists := spec.Phases[r.PhaseKey]
	if !exists {
		return fmt.Errorf("phase %s not found", r.PhaseKey)
	}

	// Checks we need:
	// 2. You can only remove future phases
	if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); !st.After(actx.CurrentTime) {
		return &subscription.PatchForbiddenError{Msg: "cannot remove already started phase"}
	}

	// And lets honor the shift behavior.
	switch r.RemoveInput.Shift {
	case subscription.RemoveSubscriptionPhaseShiftNext:
		// Let's find all subsequent phases and shift them back by the duration of the original phase
		sortedPhases := spec.GetSortedPhases()

		// We have to calculate what to shift by. Note that phase.Duration is misleading, as though it's part of the creation input, it cannot be trusted as it's only present for customizations and edits.
		deletedPhaseStart, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
		var nextPhaseStartAfter datetime.ISODuration
		for _, p := range spec.GetSortedPhases() {
			if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); v.After(deletedPhaseStart) {
				nextPhaseStartAfter = p.StartAfter
				break
			}
		}

		if nextPhaseStartAfter.IsZero() {
			// If there is no next phase then we don't need to shift anything
			break
		}

		shift, err := nextPhaseStartAfter.Subtract(phase.StartAfter)
		if err != nil {
			return fmt.Errorf("failed to calculate shift: %w", err)
		}

		reachedTargetPhase := false

		for i, p := range sortedPhases {
			if v, _ := p.StartAfter.AddTo(spec.ActiveFrom); v.After(deletedPhaseStart) {
				reachedTargetPhase = true
			}

			if reachedTargetPhase {
				sa, err := p.StartAfter.Subtract(shift)
				if err != nil {
					return fmt.Errorf("failed to shift phase %s: %w", p.PhaseKey, err)
				}
				sortedPhases[i].StartAfter = sa
			}
		}
	case subscription.RemoveSubscriptionPhaseShiftPrev:
		// We leave everything as is, the previous phase will fill up the gap
	default:
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("invalid shift behavior: %T", r.RemoveInput.Shift)}
	}

	// Then let's remove the phase
	delete(spec.Phases, r.PhaseKey)

	return nil
}
