package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PatchAddItem struct {
	PhaseKey    string
	ItemKey     string
	CreateInput subscription.SubscriptionItemSpec
}

func (a PatchAddItem) Op() subscription.PatchOperation {
	return subscription.PatchOperationAdd
}

func (a PatchAddItem) Path() subscription.SpecPath {
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
		return models.ErrorWithFieldPrefix(a.FieldDescriptor(), err)
	}

	return nil
}

func (a PatchAddItem) FieldDescriptor() *models.FieldDescriptor {
	return models.NewFieldSelectorGroup(
		models.NewFieldSelectorGroup(
			models.NewFieldSelector("phases"),
			models.NewFieldSelector(a.PhaseKey),
		).WithAttributes(models.Attributes{
			subscription.PhaseDescriptor: true,
		}),
		models.NewFieldSelector("items"),
		models.NewFieldSelector(a.ItemKey),
	)
}

func (a PatchAddItem) ValueAsAny() any {
	return a.CreateInput
}

var _ subscription.ValuePatch[subscription.SubscriptionItemSpec] = PatchAddItem{}

func (a PatchAddItem) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, ok := spec.Phases[a.PhaseKey]
	if !ok {
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("phase %s not found", a.PhaseKey)}
	}

	phaseStartTime, _ := phase.StartAfter.AddTo(spec.ActiveFrom)

	// Checks we need:

	// 1. You cannot add items to previous phases
	currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
	if !exists {
		// If the current phase doesn't exist then either all phases are in the past or in the future
		// If all phases are in the past then no addition is possible
		// If all phases are in the past then the selected one is also in the past
		if st, _ := phase.StartAfter.AddTo(spec.ActiveFrom); st.Before(actx.CurrentTime) {
			return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
		} else {
			// If it's added to a future phase, the matching key for the phase has to be empty
			if len(phase.ItemsByKey) > 0 {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to future phase %s which already has items", a.PhaseKey)}
			}
		}
	} else {
		currentPhaseStartTime, _ := currentPhase.StartAfter.AddTo(spec.ActiveFrom)

		// If the selected phase is before the current phase, it's forbidden
		if phaseStartTime.Before(currentPhaseStartTime) {
			return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which starts before current phase", a.PhaseKey)}
		} else if phase.PhaseKey == currentPhase.PhaseKey {
			// Sanity check
			if actx.CurrentTime.Before(phaseStartTime) {
				return fmt.Errorf("current time is before the current phase start which is impossible")
			}

			// 2. If it's added to the current phase, the specified start time cannot point to the past
			if a.CreateInput.ActiveFromOverrideRelativeToPhaseStart != nil {
				iST, _ := a.CreateInput.ActiveFromOverrideRelativeToPhaseStart.AddTo(phaseStartTime)
				if iST.Before(actx.CurrentTime) {
					return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which would become active in the past at %s", a.PhaseKey, iST)}
				}
			} else {
				// 3. If it's added to the current phase, and start time is not specified, it will be set for the current time, as you cannot change the past
				diff := datetime.ISODurationBetween(phaseStartTime, actx.CurrentTime)
				a.CreateInput.ActiveFromOverrideRelativeToPhaseStart = &diff
			}
		} else if phaseStartTime.After(currentPhaseStartTime) {
			// 4. If you're adding it to a future phase, the matching key for the phase has to be empty
			if len(phase.ItemsByKey[a.ItemKey]) > 0 {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to future phase %s which already has items", a.PhaseKey)}
			}
		} else {
			return fmt.Errorf("didn't enter any logical branch")
		}
	}

	// Finally, let's try to add it to the phase

	if phase.ItemsByKey[a.ItemKey] == nil {
		phase.ItemsByKey[a.ItemKey] = make([]*subscription.SubscriptionItemSpec, 0)
	}

	// If it's added to the current phase, we need to close the activity of any current item if present
	hasCurrentItemAndShouldCloseCurrentItemForKey := false

	if exists && currentPhase.PhaseKey == phase.PhaseKey {
		if len(phase.ItemsByKey[a.ItemKey]) > 0 {
			hasCurrentItemAndShouldCloseCurrentItemForKey = true
		}
	}

	if hasCurrentItemAndShouldCloseCurrentItemForKey {
		// Sanity check
		if len(phase.ItemsByKey[a.ItemKey]) == 0 {
			return fmt.Errorf("there should be an item to close")
		}

		itemToClose := phase.ItemsByKey[a.ItemKey][len(phase.ItemsByKey[a.ItemKey])-1]

		// If it already has a scheduled end time, which is later than the time this new item should start, we should error.
		// The user can circumvent this, by first issuing a delete for the item, and then adding a new one.
		if itemToClose.ActiveToOverrideRelativeToPhaseStart != nil {
			itemToCloseEndTime, _ := itemToClose.ActiveToOverrideRelativeToPhaseStart.AddTo(phaseStartTime)

			// Sanity check
			if a.CreateInput.ActiveFromOverrideRelativeToPhaseStart == nil {
				return fmt.Errorf("ActiveFromOverrideRelativeToPhaseStart should already be set when adding after an already existing item for the current phase")
			}

			itemToAddStartTime, _ := a.CreateInput.ActiveFromOverrideRelativeToPhaseStart.AddTo(phaseStartTime)

			if itemToCloseEndTime.After(itemToAddStartTime) {
				return &subscription.PatchForbiddenError{Msg: fmt.Sprintf("cannot add item to phase %s which would overlap with a current item, you should delete first", a.PhaseKey)}
			}
		}

		// Let's update the current item to close to actually close as the new item starts
		itemToClose.ActiveToOverrideRelativeToPhaseStart = a.CreateInput.ActiveFromOverrideRelativeToPhaseStart
	}

	// Finally, we simply add it as the last Spec for its key in the phase

	phase.ItemsByKey[a.ItemKey] = append(phase.ItemsByKey[a.ItemKey], &a.CreateInput)
	return nil
}
