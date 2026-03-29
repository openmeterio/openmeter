package reconciler

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/chargeupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type usageBasedChargeCollection struct {
	chargePatchCollection
}

func newUsageBasedChargeCollection(preallocatedCapacity int) *usageBasedChargeCollection {
	return &usageBasedChargeCollection{
		chargePatchCollection: newChargePatchCollection(persistedstate.ItemTypeChargeUsageBased, preallocatedCapacity),
	}
}

func (c *usageBasedChargeCollection) AddCreate(target targetstate.StateItem) error {
	rateCardMeta := target.Spec.RateCard.AsMeta()
	price := rateCardMeta.Price
	if price == nil {
		return fmt.Errorf("price is required for usage based charge")
	}

	baseIntent, err := newChargeIntentBaseFromTargetState(target)
	if err != nil {
		return err
	}

	intent := charges.NewChargeIntent(chargesusagebased.Intent{
		Intent:         baseIntent,
		InvoiceAt:      target.GetInvoiceAt(),
		SettlementMode: target.Subscription.SettlementMode,
		FeatureKey:     lo.FromPtr(rateCardMeta.FeatureKey),
		Price:          *price,
		Discounts:      rateCardMeta.Discounts,
	})

	return c.addPatch(target.UniqueID, PatchOperationCreate, chargeupdater.NewCreatePatch(intent))
}

func (c *usageBasedChargeCollection) AddDelete(uniqueID string, existing persistedstate.Item) error {
	return c.unsupportedOperationError(PatchOperationDelete, uniqueID, existing)
}

func (c *usageBasedChargeCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	return c.unsupportedOperationError(PatchOperationShrink, uniqueID, existing)
}

func (c *usageBasedChargeCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	return c.unsupportedOperationError(PatchOperationExtend, target.UniqueID, existing)
}

func (c *usageBasedChargeCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	return c.unsupportedOperationError(PatchOperationProrate, target.UniqueID, existing)
}
