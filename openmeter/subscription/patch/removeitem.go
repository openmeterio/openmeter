package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
)

type PatchRemoveItem struct {
	PhaseKey string
	ItemKey  string
}

func (r PatchRemoveItem) Op() subscription.PatchOperation {
	return subscription.PatchOperationRemove
}

func (r PatchRemoveItem) Path() subscription.PatchPath {
	return subscription.NewItemPath(r.PhaseKey, r.ItemKey)
}

func (r PatchRemoveItem) Validate() error {
	if err := r.Path().Validate(); err != nil {
		return err
	}

	if err := r.Op().Validate(); err != nil {
		return err
	}

	return nil
}

var _ subscription.Patch = PatchRemoveItem{}

func (r PatchRemoveItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, ok := spec.Phases[r.PhaseKey]
	if !ok {
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("phase %s not found", r.PhaseKey)}
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(spec.ActiveFrom)

	if items, exists := phase.ItemsByKey[r.ItemKey]; !exists || len(items) == 0 {
		return &subscription.PatchConflictError{Msg: fmt.Sprintf("items for key %s doesn't exists in phase %s", r.ItemKey, r.PhaseKey)}
	}

	// Checks we need:
	// 1. You cannot remove items from previous phases
	if actx.Operation == subscription.SpecOperationEdit {
		currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
		if !exists {
			// either all phases are in the past or in the future
			// if all phases are in the past then no removal is possible
			//
			// If all phases are in the past then the selected one is also in the past
			if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); st.Before(actx.CurrentTime) {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot remove item from phase %s which starts before current phase", r.PhaseKey)}
			}
		} else {
			currentPhaseStartTime, _ := currentPhase.StartAfter.AddTo(spec.ActiveFrom)
			if phaseStartTime.Before(currentPhaseStartTime) {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot remove item from phase %s which starts before current phase", r.PhaseKey)}
			}
		}
	}

	// Finally, lets try to remove the item
	currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
	if exists && currentPhase.PhaseKey == r.PhaseKey {
		// If it's removed from the current phase, we should set its end time to the current time, instead of deleting it (as we cannot falsify history)

		diff := datex.Between(phaseStartTime, actx.CurrentTime)

		phase.ItemsByKey[r.ItemKey][len(phase.ItemsByKey[r.ItemKey])-1].ActiveToOverrideRelativeToPhaseStart = &diff
	} else {
		// Otherwise (if its a future phase), we can just remove it
		phase.ItemsByKey[r.ItemKey] = phase.ItemsByKey[r.ItemKey][:len(phase.ItemsByKey[r.ItemKey])-1]

		// And let's clean up the items array if it's empty
		if len(phase.ItemsByKey[r.ItemKey]) == 0 {
			delete(phase.ItemsByKey, r.ItemKey)
		}
	}

	return nil
}
