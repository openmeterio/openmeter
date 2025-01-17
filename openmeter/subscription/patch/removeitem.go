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
	phase, rel, err := phaseContentHelper{spec: *spec, actx: actx}.GetPhaseForEdit(r.PhaseKey)
	if err != nil {
		return err
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(spec.ActiveFrom)

	if items, exists := phase.ItemsByKey[r.ItemKey]; !exists || len(items) == 0 {
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("items for key %s doesn't exists in phase %s", r.ItemKey, r.PhaseKey)}
	}

	// Finally, lets try to remove the item
	if rel == isCurrentPhase {
		// If it's removed from the current phase, we should set its end time to the current time, instead of deleting it (as we cannot falsify history)

		diff := datex.Between(phaseStartTime, actx.CurrentTime)

		phase.ItemsByKey[r.ItemKey][len(phase.ItemsByKey[r.ItemKey])-1].CadenceOverrideRelativeToPhaseStart.ActiveToOverride = &diff
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
