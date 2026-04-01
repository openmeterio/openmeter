package reconciler

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
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

	return c.addCreate(intent)
}
