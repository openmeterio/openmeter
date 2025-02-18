package patch

import (
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type PatchUnscheduleEdit struct{}

func (p PatchUnscheduleEdit) Op() subscription.PatchOperation {
	return subscription.PatchOperationUnschedule
}

func (p PatchUnscheduleEdit) Path() subscription.PatchPath {
	return subscription.NewPhasePath("")
}

func (p PatchUnscheduleEdit) Validate() error {
	if err := p.Path().Validate(); err != nil {
		return err
	}

	if err := p.Op().Validate(); err != nil {
		return err
	}

	return nil
}

var _ subscription.Patch = PatchUnscheduleEdit{}

// "Unscheduling an edit" is a concept that might intuitively makes sense for clients:
// 1. Making some edit to a subscription
// 2. You want to do another edit
// 3. Your edit discards the previous edit
//
// However, this is not really a behavior that makes sense from a server perspective.
// The compromise we make is that, for most cases, when a client wants to unschedule an edit,
// all they really want to do is get rid of any future changes to the subscription.
//
// As editing future phases doesn't create multiple versions of the items (see AddItem and RemoveItem),
// we only tackle editing the current phase.
//
// UnscheduleEdit simply removes all scheduled versions of the items in the current phase.
func (p PatchUnscheduleEdit) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	currentPhase, exists := spec.GetCurrentPhaseAt(actx.CurrentTime)
	if !exists {
		return &subscription.PatchConflictError{Msg: "current phase doesn't exist, cannot unschedule edits"}
	}

	currentPhaseCadence, err := spec.GetPhaseCadence(currentPhase.PhaseKey)
	if err != nil {
		return err
	}

	for iK, items := range currentPhase.ItemsByKey {
		for i := len(items) - 1; i >= 0; i-- {
			if c := items[i].GetCadence(currentPhaseCadence); c.IsActiveAt(actx.CurrentTime) {
				// Let's make sure there is no scheduled end time
				currentPhase.ItemsByKey[iK][i].ActiveToOverrideRelativeToPhaseStart = nil

				break
			}

			// Let's remove the item at index i from the array
			currentPhase.ItemsByKey[iK] = items[:i]
		}
	}

	return nil
}
