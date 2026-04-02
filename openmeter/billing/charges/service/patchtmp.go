package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

// tmpMapShrinkExtendToCreateDelete is a temporary function to map shrink/extend patches to a sequence of deleting
// the old charge and creating a new charge.
//
// This is a temporary function so that we don't have to implement the shrink/extend logic just yet. When we are to
// implement the credit_then_invoice logic we need to have a proper shrink/extend patching in place.
func (s *service) tmpMapShrinkExtendToCreateDelete(ctx context.Context, input charges.ApplyPatchesInput) (charges.ApplyPatchesInput, error) {
	chargeIDsToReplace := make([]string, 0, len(input.PatchesByChargeID))
	out := charges.ApplyPatchesInput{
		CustomerID:        input.CustomerID,
		Creates:           append(make(charges.ChargeIntents, 0, len(input.Creates)), input.Creates...),
		PatchesByChargeID: make(map[string]charges.Patch, len(input.PatchesByChargeID)),
	}

	for chargeID, patch := range input.PatchesByChargeID {
		switch patch.(type) {
		case meta.PatchShrink, meta.PatchExtend:
			out.PatchesByChargeID[chargeID] = meta.PatchDelete{Policy: meta.RefundAsCreditsDeletePolicy}
			chargeIDsToReplace = append(chargeIDsToReplace, chargeID)
		default:
			out.PatchesByChargeID[chargeID] = patch
		}
	}

	if len(chargeIDsToReplace) == 0 {
		return out, nil
	}

	chargeSearchItems, err := s.adapter.GetByIDs(ctx, charges.GetByIDsInput{
		Namespace: input.CustomerID.Namespace,
		IDs:       chargeIDsToReplace,
	})
	if err != nil {
		return charges.ApplyPatchesInput{}, fmt.Errorf("getting charges for shrink/extend remap: %w", err)
	}

	existingCharges, err := s.expandChargesWithTypes(ctx, input.CustomerID.Namespace, chargeSearchItems, meta.ExpandNone)
	if err != nil {
		return charges.ApplyPatchesInput{}, fmt.Errorf("expanding charges for shrink/extend remap: %w", err)
	}

	existingByID := lo.SliceToMap(existingCharges, func(charge charges.Charge) (string, charges.Charge) {
		return charge.GetID(), charge
	})

	for _, chargeID := range chargeIDsToReplace {
		patch := input.PatchesByChargeID[chargeID]
		existingCharge, ok := existingByID[chargeID]
		if !ok {
			return charges.ApplyPatchesInput{}, fmt.Errorf("charge %s not found for shrink/extend remap", chargeID)
		}

		intent, err := tmpRemapShrinkExtendToCreateIntent(existingCharge, patch)
		if err != nil {
			return charges.ApplyPatchesInput{}, fmt.Errorf("remapping charge %s shrink/extend to create intent: %w", chargeID, err)
		}

		out.Creates = append(out.Creates, intent)
	}

	return out, nil
}

func tmpRemapShrinkExtendToCreateIntent(existing charges.Charge, patch charges.Patch) (charges.ChargeIntent, error) {
	switch typedPatch := patch.(type) {
	case meta.PatchShrink:
		existingIntent, err := tmpChargeMetaIntent(existing)
		if err != nil {
			return charges.ChargeIntent{}, err
		}

		if err := typedPatch.ValidateWith(existingIntent); err != nil {
			return charges.ChargeIntent{}, fmt.Errorf("validating shrink patch: %w", err)
		}

		return tmpApplyPatchToCreateIntent(existing, typedPatch.NewServicePeriodTo, typedPatch.NewFullServicePeriodTo, typedPatch.NewBillingPeriodTo)
	case meta.PatchExtend:
		existingIntent, err := tmpChargeMetaIntent(existing)
		if err != nil {
			return charges.ChargeIntent{}, err
		}

		if err := typedPatch.ValidateWith(existingIntent); err != nil {
			return charges.ChargeIntent{}, fmt.Errorf("validating extend patch: %w", err)
		}

		return tmpApplyPatchToCreateIntent(existing, typedPatch.NewServicePeriodTo, typedPatch.NewFullServicePeriodTo, typedPatch.NewBillingPeriodTo)
	default:
		return charges.ChargeIntent{}, fmt.Errorf("unsupported patch type for shrink/extend remap: %T", patch)
	}
}

func tmpChargeMetaIntent(existing charges.Charge) (meta.Intent, error) {
	switch existing.Type() {
	case meta.ChargeTypeFlatFee:
		charge, err := existing.AsFlatFeeCharge()
		if err != nil {
			return meta.Intent{}, err
		}

		return charge.Intent.Intent, nil
	case meta.ChargeTypeUsageBased:
		charge, err := existing.AsUsageBasedCharge()
		if err != nil {
			return meta.Intent{}, err
		}

		return charge.Intent.Intent, nil
	default:
		return meta.Intent{}, fmt.Errorf("unsupported charge type for shrink/extend validation: %s", existing.Type())
	}
}

func tmpApplyPatchToCreateIntent(existing charges.Charge, newServicePeriodTo, newFullServicePeriodTo, newBillingPeriodTo time.Time) (charges.ChargeIntent, error) {
	switch existing.Type() {
	case meta.ChargeTypeFlatFee:
		charge, err := existing.AsFlatFeeCharge()
		if err != nil {
			return charges.ChargeIntent{}, err
		}

		intent := charge.Intent
		intent.ServicePeriod.To = newServicePeriodTo
		intent.FullServicePeriod.To = newFullServicePeriodTo
		intent.BillingPeriod.To = newBillingPeriodTo
		intent = intent.Normalized()

		return charges.NewChargeIntent(intent), nil
	case meta.ChargeTypeUsageBased:
		charge, err := existing.AsUsageBasedCharge()
		if err != nil {
			return charges.ChargeIntent{}, err
		}

		intent := charge.Intent
		intent.ServicePeriod.To = newServicePeriodTo
		intent.FullServicePeriod.To = newFullServicePeriodTo
		intent.BillingPeriod.To = newBillingPeriodTo
		intent = intent.Normalized()

		return charges.NewChargeIntent(intent), nil
	default:
		return charges.ChargeIntent{}, fmt.Errorf("unsupported charge type for shrink/extend remap: %s", existing.Type())
	}
}
