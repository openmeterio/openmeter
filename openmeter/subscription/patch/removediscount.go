package patch

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/defaultx"
)

type PatchRemoveDiscount struct {
	PhaseKey string

	// Which item should be removed (defined by its index)
	RemoveAtIdx int

	// At what time the Discounts are queried (determines indexing). Defaults to now.
	// Explanation:
	// A and B Discounts are already active, C is set to activate in the future and would be ordered A C B.
	// If we want to remove the discount at idx 1, we have to disambiguate between C and B, and for that we use this timestamp.
	IndexTime *time.Time
}

func (a PatchRemoveDiscount) Op() subscription.PatchOperation {
	return subscription.PatchOperationRemove
}

func (a PatchRemoveDiscount) Path() subscription.PatchPath {
	return subscription.NewItemPath(a.PhaseKey, fmt.Sprintf("%d", a.RemoveAtIdx))
}

func (a PatchRemoveDiscount) Validate() error {
	if err := a.Path().Validate(); err != nil {
		return err
	}

	if err := a.Op().Validate(); err != nil {
		return err
	}

	if a.RemoveAtIdx < 0 {
		return &subscription.PatchValidationError{Msg: "removeAt must be a non-negative integer"}
	}

	return nil
}

func (a PatchRemoveDiscount) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, rel, err := phaseContentHelper{spec: *spec, actx: actx}.GetPhaseForEdit(a.PhaseKey)
	if err != nil {
		return err
	}

	phaseCadence, err := spec.GetPhaseCadence(a.PhaseKey)
	if err != nil {
		return err
	}

	// Now let's find which index should be removed
	filterTime := defaultx.WithDefault(a.IndexTime, actx.CurrentTime)
	if rel == isFuturePhase {
		filterTime = phaseCadence.ActiveFrom
	}

	// Now let's find the active items at that time
	var activeDiscounts []struct {
		originalIdx int
		spec        subscription.DiscountSpec
	}

	for i, d := range phase.Discounts {
		cadence := d.CadenceOverrideRelativeToPhaseStart.GetCadence(phaseCadence)
		if cadence.IsActiveAt(filterTime) {
			activeDiscounts = append(activeDiscounts, struct {
				originalIdx int
				spec        subscription.DiscountSpec
			}{originalIdx: i, spec: d})
		}
	}

	// Now let's find the index of the discount to remove
	if a.RemoveAtIdx >= len(activeDiscounts) {
		return &subscription.PatchValidationError{Msg: fmt.Sprintf("index %d out of bounds for %d items", a.RemoveAtIdx, len(activeDiscounts))}
	}

	indexToRemove := activeDiscounts[a.RemoveAtIdx].originalIdx

	// Now let's remove the discount
	// If the discount to remove is already active we mark it as inactive starting now
	if rel == isCurrentPhase {
		diff := datex.Between(phaseCadence.ActiveFrom, actx.CurrentTime)

		phase.Discounts[indexToRemove].ActiveToOverride = &diff
	} else {
		// Otherwise (if its a future phase), we can just remove it
		phase.Discounts = append(phase.Discounts[:indexToRemove], phase.Discounts[indexToRemove+1:]...)
	}

	return nil
}

var _ subscription.Patch = PatchRemoveDiscount{}
