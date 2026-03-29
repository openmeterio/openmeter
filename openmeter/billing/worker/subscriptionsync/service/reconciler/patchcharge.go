package reconciler

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/chargeupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type chargePatchCollection struct {
	itemType persistedstate.ItemType
	patches  []ChargePatch
}

func newChargePatchCollection(itemType persistedstate.ItemType, preallocatedCapacity int) chargePatchCollection {
	if preallocatedCapacity <= 0 {
		preallocatedCapacity = 16
	}

	return chargePatchCollection{
		itemType: itemType,
		patches:  make([]ChargePatch, 0, preallocatedCapacity),
	}
}

func (c *chargePatchCollection) addPatch(uniqueID string, operation PatchOperation, updaterPatch chargeupdater.Patch) error {
	if uniqueID == "" {
		return fmt.Errorf("unique id is required [operation=%s, item_type=%s]", operation, c.itemType)
	}

	c.patches = append(c.patches, newGenericChargePatch(uniqueID, operation, c.itemType, updaterPatch))

	return nil
}

func (c chargePatchCollection) unsupportedOperationError(operation PatchOperation, uniqueID string, existing persistedstate.Item) error {
	return fmt.Errorf("unsupported operation %s for charge patches [item_type=%s, uniqueID=%s, id=%s]", operation, c.itemType, uniqueID, existing.ID())
}

func (c chargePatchCollection) IsEmpty() bool {
	return len(c.patches) == 0
}

func (c chargePatchCollection) Patches() []ChargePatch {
	return c.patches
}

type genericChargePatch struct {
	uniqueID     string
	operation    PatchOperation
	itemType     persistedstate.ItemType
	updaterPatch chargeupdater.Patch
}

func newGenericChargePatch(uniqueID string, operation PatchOperation, itemType persistedstate.ItemType, updaterPatch chargeupdater.Patch) genericChargePatch {
	return genericChargePatch{
		uniqueID:     uniqueID,
		operation:    operation,
		itemType:     itemType,
		updaterPatch: updaterPatch,
	}
}

func (p genericChargePatch) Operation() PatchOperation {
	return p.operation
}

func (p genericChargePatch) UniqueReferenceID() string {
	return p.uniqueID
}

func (p genericChargePatch) GetChargePatch() chargeupdater.Patch {
	return p.updaterPatch
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
