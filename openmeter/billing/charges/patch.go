package charges

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Patch = meta.Patch

var _ models.Validator = (*ApplyPatchesInput)(nil)

type ApplyPatchesInput struct {
	CustomerID customer.CustomerID
	Creates    ChargeIntents

	// PatchesByChargeID is a map of charge ID to the patches to apply to the charge. This format is used to make sure
	// there's only a single patch affecting a single charge.
	PatchesByChargeID map[string]Patch
}

func (i ApplyPatchesInput) Validate() error {
	var errs []error
	if err := i.CustomerID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	if err := i.Creates.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("creates: %w", err))
	}

	for chargeID, patch := range i.PatchesByChargeID {
		if chargeID == "" {
			errs = append(errs, fmt.Errorf("charge ID is required"))
			continue
		}

		if patch == nil {
			errs = append(errs, fmt.Errorf("patch for charge ID %s is nil", chargeID))
			continue
		}

		if err := patch.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("patch for charge ID %s: %w", chargeID, err))
		}
	}

	return errors.Join(errs...)
}

// ConcatenateApplyPatchesInputs concatenates the given inputs into a single input, while enforcing uniqueness constraints.
func ConcatenateApplyPatchesInputs(inputs ...ApplyPatchesInput) (ApplyPatchesInput, error) {
	result := ApplyPatchesInput{
		CustomerID:        inputs[0].CustomerID,
		Creates:           make(ChargeIntents, 0, lo.SumBy(inputs, func(input ApplyPatchesInput) int { return len(input.Creates) })),
		PatchesByChargeID: make(map[string]Patch, lo.SumBy(inputs, func(input ApplyPatchesInput) int { return len(input.PatchesByChargeID) })),
	}

	for _, input := range inputs {
		result.Creates = append(result.Creates, input.Creates...)
		for chargeID, patch := range input.PatchesByChargeID {
			if _, exists := result.PatchesByChargeID[chargeID]; exists {
				return ApplyPatchesInput{}, fmt.Errorf("duplicate charge ID: %s", chargeID)
			}
			result.PatchesByChargeID[chargeID] = patch
		}
	}

	return result, nil
}

func (i ApplyPatchesInput) IsEmpty() bool {
	return len(i.PatchesByChargeID) == 0 && len(i.Creates) == 0
}
