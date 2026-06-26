package reconciler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type chargePatchCollection struct {
	engineType billing.LineEngineType
	itemType   persistedstate.ItemType
	patches    charges.ApplyPatchesInput
}

func (c chargePatchCollection) GetLineEngineType() billing.LineEngineType {
	return c.engineType
}

func newChargePatchCollection(engineType billing.LineEngineType, itemType persistedstate.ItemType, preallocatedCapacity int) chargePatchCollection {
	if preallocatedCapacity <= 0 {
		preallocatedCapacity = 16
	}

	return chargePatchCollection{
		engineType: engineType,
		itemType:   itemType,
		patches: charges.ApplyPatchesInput{
			PatchesByChargeID: make(map[string]charges.Patch, preallocatedCapacity),
			Creates:           make(charges.ChargeIntents, 0, preallocatedCapacity),
		},
	}
}

func (c chargePatchCollection) unsupportedOperationError(operation PatchOperation, uniqueID string, existing persistedstate.Item) error {
	return fmt.Errorf("unsupported operation %s for charge patches [item_type=%s, uniqueID=%s, id=%s]", operation, c.itemType, uniqueID, existing.ID())
}

func (c chargePatchCollection) IsEmpty() bool {
	return len(c.patches.PatchesByChargeID) == 0 && len(c.patches.Creates) == 0
}

func (c chargePatchCollection) Patches() charges.ApplyPatchesInput {
	return c.patches
}

func (c *chargePatchCollection) addCreate(intent charges.ChargeIntent) error {
	// Full intent validation is intentionally delayed until charges.Service.ApplyPatches,
	// after namespace default tax codes are applied to create intents.
	uniqueReferenceID, err := intent.GetUniqueReferenceID()
	if err != nil {
		return fmt.Errorf("getting unique reference ID: %w", err)
	}

	if lo.FromPtr(uniqueReferenceID) == "" {
		return fmt.Errorf("unique reference ID is required")
	}

	c.patches.Creates = append(c.patches.Creates, intent)
	return nil
}

func (c *chargePatchCollection) addPatch(chargeID string, patch charges.Patch) error {
	if chargeID == "" {
		return fmt.Errorf("charge ID is required")
	}

	if patch == nil {
		return fmt.Errorf("patch is required")
	}

	if err := patch.Validate(); err != nil {
		return fmt.Errorf("invalid patch: %w", err)
	}

	if _, exists := c.patches.PatchesByChargeID[chargeID]; exists {
		return fmt.Errorf("patch for charge ID %s already exists", chargeID)
	}

	c.patches.PatchesByChargeID[chargeID] = patch
	return nil
}

func (c *chargePatchCollection) AddDelete(_ string, existing persistedstate.Item) error {
	patch, err := chargesmeta.NewPatchDelete(chargesmeta.NewPatchDeleteInput{
		Target: chargesmeta.ChangeTargetBase,
		Policy: chargesmeta.RefundAsCreditsDeletePolicy,
	})
	if err != nil {
		return err
	}

	return c.addPatch(existing.ID().ID, patch)
}

func (c *chargePatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	// Charge-backed reconciliation does not emit explicit prorate patches. For charges,
	// any period-shape change is carried by shrink/extend and the charge domain is
	// responsible for recalculating the effective amount from the updated periods.
	return c.unsupportedOperationError(PatchOperationProrate, target.UniqueID, existing)
}

func (c *chargePatchCollection) addEmulatedReplacement(existing persistedstate.Item, replacement charges.ChargeIntent) error {
	deletePatch, err := chargesmeta.NewPatchDelete(chargesmeta.NewPatchDeleteInput{
		Target: chargesmeta.ChangeTargetBase,
		Policy: chargesmeta.RefundAsCreditsDeletePolicy,
	})
	if err != nil {
		return fmt.Errorf("creating replacement delete patch: %w", err)
	}

	if err := c.addPatch(existing.ID().ID, deletePatch); err != nil {
		return fmt.Errorf("adding replacement delete patch: %w", err)
	}

	if err := c.addCreate(replacement); err != nil {
		return fmt.Errorf("adding replacement create intent: %w", err)
	}

	return nil
}

func logChargesPatches(ctx context.Context, log *slog.Logger, patches charges.ApplyPatchesInput) {
	for chargeID, patch := range patches.PatchesByChargeID {
		log.InfoContext(ctx, "patching charge", "charge_id", chargeID, "patch", patch)
	}

	for _, intent := range patches.Creates {
		log.InfoContext(ctx, "creating charge", "intent", intent)
	}
}
