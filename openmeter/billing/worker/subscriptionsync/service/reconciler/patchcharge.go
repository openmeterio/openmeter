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
	itemType persistedstate.ItemType
	patches  charges.ApplyPatchesInput
}

func (c chargePatchCollection) GetBackendType() BackendType {
	return BackendTypeCharges
}

func newChargePatchCollection(itemType persistedstate.ItemType, preallocatedCapacity int) chargePatchCollection {
	if preallocatedCapacity <= 0 {
		preallocatedCapacity = 16
	}

	return chargePatchCollection{
		itemType: itemType,
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
	if intent.Validate() != nil {
		return fmt.Errorf("invalid intent: %w", intent.Validate())
	}

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
	return c.addPatch(existing.ID().ID, chargesmeta.PatchDelete{
		Policy: chargesmeta.RefundAsCreditsDeletePolicy,
	})
}

func (c *chargePatchCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	targetServicePeriod := target.GetServicePeriod()

	return c.addPatch(existing.ID().ID, chargesmeta.PatchShrink{
		NewServicePeriodTo:     targetServicePeriod.To,
		NewFullServicePeriodTo: target.FullServicePeriod.End,
		NewBillingPeriodTo:     target.BillingPeriod.End,
	})
}

func (c *chargePatchCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	targetServicePeriod := target.GetServicePeriod()

	return c.addPatch(existing.ID().ID, chargesmeta.PatchExtend{
		NewServicePeriodTo:     targetServicePeriod.To,
		NewFullServicePeriodTo: target.FullServicePeriod.End,
		NewBillingPeriodTo:     target.BillingPeriod.End,
	})
}

func (c *chargePatchCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	// Charge-backed reconciliation does not emit explicit prorate patches. For charges,
	// any period-shape change is carried by shrink/extend and the charge domain is
	// responsible for recalculating the effective amount from the updated periods.
	return c.unsupportedOperationError(PatchOperationProrate, target.UniqueID, existing)
}

func newChargeIntentBaseFromTargetState(target targetstate.StateItem) (chargesmeta.Intent, error) {
	rateCardMeta := target.Spec.RateCard.AsMeta()
	annotations, err := target.SubscriptionItem.Annotations.Clone()
	if err != nil {
		return chargesmeta.Intent{}, fmt.Errorf("cloning annotations: %w", err)
	}

	return chargesmeta.Intent{
		Name:          rateCardMeta.Name,
		Description:   rateCardMeta.Description,
		Metadata:      target.SubscriptionItem.Metadata.Clone(),
		Annotations:   annotations,
		ManagedBy:     billing.SubscriptionManagedLine,
		CustomerID:    target.Subscription.CustomerId,
		Currency:      target.CurrencyCalculator.Currency,
		ServicePeriod: target.GetServicePeriod(),
		FullServicePeriod: timeutil.ClosedPeriod{
			From: target.FullServicePeriod.Start,
			To:   target.FullServicePeriod.End,
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: target.BillingPeriod.Start,
			To:   target.BillingPeriod.End,
		},
		TaxConfig:         rateCardMeta.TaxConfig,
		UniqueReferenceID: &target.UniqueID,
		Subscription: &chargesmeta.SubscriptionReference{
			SubscriptionID: target.Subscription.ID,
			PhaseID:        target.PhaseID,
			ItemID:         target.SubscriptionItem.ID,
		},
	}, nil
}

func logChargesPatches(ctx context.Context, log *slog.Logger, patches charges.ApplyPatchesInput) {
	for chargeID, patch := range patches.PatchesByChargeID {
		log.Info("patching charge", "charge_id", chargeID, "patch", patch)
	}

	for chargeID, patch := range patches.Creates {
		log.Info("creating charge", "charge_id", chargeID, "patch", patch)
	}
}
