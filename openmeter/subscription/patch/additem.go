package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput subscription.SubscriptionItemSpec
}

func (a PatchAddItem) Op() subscription.PatchOperation {
	return subscription.PatchOperationAdd
}

func (a PatchAddItem) Path() subscription.PatchPath {
	return subscription.NewItemPath(a.PhaseKey, a.ItemKey)
}

func (a PatchAddItem) Value() subscription.SubscriptionItemSpec {
	return a.CreateInput
}

func (a PatchAddItem) Validate() error {
	if err := a.Path().Validate(); err != nil {
		return err
	}

	if err := a.Op().Validate(); err != nil {
		return err
	}

	if err := a.CreateInput.Validate(); err != nil {
		return err
	}

	return nil
}

var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}

func (a PatchAddItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, rel, err := phaseContentHelper{spec: *spec, actx: actx}.GetPhaseForEdit(a.PhaseKey)
	if err != nil {
		return err
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(spec.ActiveFrom)

	cadenceHelper := relativeCadenceHelper{
		contentType:    "item",
		phaseStartTime: phaseStartTime,
		phaseKey:       a.PhaseKey,
		rel:            rel,
		actx:           actx,
	}
	if err := cadenceHelper.ValidateRelativeCadence(&a.CreateInput.CadenceOverrideRelativeToPhaseStart); err != nil {
		return err
	}

	// If you're adding it to a future phase, the matching key for the phase has to be empty
	if rel == isFuturePhase {
		if len(phase.ItemsByKey[a.ItemKey]) > 0 {
			return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to future phase %s which already has items", a.PhaseKey)}
		}
	}

	// Finally, let's try to add it to the phase

	if phase.ItemsByKey[a.ItemKey] == nil {
		phase.ItemsByKey[a.ItemKey] = make([]*subscription.SubscriptionItemSpec, 0)
	}

	// If it's added to the current phase, we need to close the activity of any current item if present
	hasCurrentItemAndShouldCloseCurrentItemForKey := false

	if rel == isCurrentPhase {
		if len(phase.ItemsByKey[a.ItemKey]) > 0 {
			hasCurrentItemAndShouldCloseCurrentItemForKey = true
		}
	}

	if hasCurrentItemAndShouldCloseCurrentItemForKey {
		// Sanity check
		if len(phase.ItemsByKey[a.ItemKey]) == 0 {
			return fmt.Errorf("There should be an item to close")
		}

		itemToClose := phase.ItemsByKey[a.ItemKey][len(phase.ItemsByKey[a.ItemKey])-1]

		// If it already has a scheduled end time, which is later than the time this new item should start, we should error.
		// The user can circumvent this, by first issuing a delete for the item, and then adding a new one.
		if itemToClose.CadenceOverrideRelativeToPhaseStart.ActiveToOverride != nil {
			itemToCloseEndTime, _ := itemToClose.CadenceOverrideRelativeToPhaseStart.ActiveToOverride.AddTo(phaseStartTime)

			// Sanity check
			if a.CreateInput.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride == nil {
				return fmt.Errorf("ActiveFromOverrideRelativeToPhaseStart should already be set when adding after an already existing item for the current phase")
			}

			itemToAddStartTime, _ := a.CreateInput.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride.AddTo(phaseStartTime)

			if itemToCloseEndTime.After(itemToAddStartTime) {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which would overlap with a current item, you should delete first", a.PhaseKey)}
			}
		}

		// Let's update the current item to close to actually close as the new item starts
		itemToClose.CadenceOverrideRelativeToPhaseStart.ActiveToOverride = a.CreateInput.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride
	}

	// Finally, we simply add it as the last Spec for its key in the phase

	phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)
	return nil
}
