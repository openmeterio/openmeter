package patch

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type PatchAddDiscount struct {
	PhaseKey    string
	InsertAt    int
	CreateInput subscription.DiscountSpec
}

func (a PatchAddDiscount) Op() subscription.PatchOperation {
	return subscription.PatchOperationAdd
}

func (a PatchAddDiscount) Path() subscription.PatchPath {
	return subscription.NewItemPath(a.PhaseKey, fmt.Sprintf("%d", a.InsertAt))
}

func (a PatchAddDiscount) Value() subscription.DiscountSpec {
	return a.CreateInput
}

func (a PatchAddDiscount) Validate() error {
	if err := a.Path().Validate(); err != nil {
		return err
	}

	if err := a.Op().Validate(); err != nil {
		return err
	}

	if err := a.CreateInput.Validate(); err != nil {
		return err
	}

	if a.CreateInput.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride != nil ||
		a.CreateInput.CadenceOverrideRelativeToPhaseStart.ActiveToOverride != nil {
		return &subscription.PatchValidationError{Msg: "cannot set active times for discount"}
	}

	if a.InsertAt < 0 {
		return &subscription.PatchValidationError{Msg: "insertAt must be a non-negative integer"}
	}

	return nil
}

func (a PatchAddDiscount) ApplyTo(spec *subscription.SubscriptionSpec, actx subscription.ApplyContext) error {
	phase, rel, err := phaseContentHelper{spec: *spec, actx: actx}.GetPhaseForEdit(a.PhaseKey)
	if err != nil {
		return err
	}

	phaseCadence, err := spec.GetPhaseCadence(a.PhaseKey)
	if err != nil {
		return err
	}

	// Let's check that all items do exist
	for _, k := range a.CreateInput.Discount.RateCardKeys() {
		if _, ok := phase.ItemsByKey[k]; !ok {
			return &subscription.PatchConflictError{Msg: fmt.Sprintf("item %s not found", k)}
		}
	}

	// Let's check the relative cadence
	cadenceHelper := relativeCadenceHelper{
		contentType:    "discount",
		phaseStartTime: phaseCadence.ActiveFrom,
		phaseKey:       a.PhaseKey,
		rel:            rel,
		actx:           actx,
	}
	if err := cadenceHelper.ValidateRelativeCadence(&a.CreateInput.CadenceOverrideRelativeToPhaseStart); err != nil {
		return err
	}

	// Let's get when the discount activates
	discountCadence := a.CreateInput.CadenceOverrideRelativeToPhaseStart.GetCadence(phaseCadence)

	// Now let's find the list of active discounts at that time
	// This is because the index is relative to that slice, explanation:
	// The Discount becomes active at T1 (deterministic from discount and spec)
	// You want the discount to be the indexAt-th active discount at T1
	var activeDiscounts []struct {
		originalIdx int
		spec        subscription.DiscountSpec
	}

	for i, d := range phase.Discounts {
		cadence := d.CadenceOverrideRelativeToPhaseStart.GetCadence(phaseCadence)
		if cadence.IsActiveAt(discountCadence.ActiveFrom) {
			activeDiscounts = append(activeDiscounts, struct {
				originalIdx int
				spec        subscription.DiscountSpec
			}{originalIdx: i, spec: d})
		}
	}

	// Now let's find which index the discount should be inserted at and insert it
	// If there is an element at that index in the filtered list, it should be inserted right before that
	// If there isn't, it should be inserted at the end

	var insertIndex int

	// If it's later than all active discounts, we just append it
	if len(activeDiscounts) == 0 {
		insertIndex = 0
	} else if a.InsertAt >= len(activeDiscounts) {
		insertIndex = activeDiscounts[len(activeDiscounts)-1].originalIdx + 1
	} else {
		insertIndex = activeDiscounts[a.InsertAt].originalIdx
	}

	// Now let's insert the discount.
	// The discount at the provided index (and all later ones) will be shifted to the right
	discounts := make([]subscription.DiscountSpec, len(phase.Discounts)+1)
	for i := 0; i < len(phase.Discounts); i++ {
		if i < insertIndex {
			discounts[i] = phase.Discounts[i]
		} else {
			discounts[i+1] = phase.Discounts[i]
		}
	}
	discounts[insertIndex] = a.CreateInput

	phase.Discounts = discounts

	return nil
}

var _ subscription.ValuePatch[subscription.DiscountSpec] = PatchAddDiscount{}
