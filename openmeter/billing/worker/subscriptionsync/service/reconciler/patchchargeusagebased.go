package reconciler

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargesusagebased "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type usageBasedChargeCollection struct {
	chargePatchCollection
}

func newUsageBasedChargeCollection(preallocatedCapacity int) *usageBasedChargeCollection {
	return &usageBasedChargeCollection{
		chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeChargeUsageBased, persistedstate.ItemTypeChargeUsageBased, preallocatedCapacity),
	}
}

func (c *usageBasedChargeCollection) AddCreate(target targetstate.StateItem) error {
	intent, err := newUsageBasedChargeIntent(target)
	if err != nil {
		return err
	}

	return c.addCreate(intent)
}

func (c *usageBasedChargeCollection) AddShrink(_ string, existing persistedstate.Item, target targetstate.StateItem) error {
	existingCharge, ok := existing.(persistedstate.UsageBasedChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a usage based charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	if existingCharge.GetUsageBasedCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newUsageBasedChargeIntent(target)
		if err != nil {
			return err
		}

		return c.addEmulatedReplacement(existing, intent)
	}

	targetServicePeriod := target.GetServicePeriod()

	patch, err := chargesmeta.NewPatchShrink(chargesmeta.NewPatchShrinkInput{
		NewServicePeriodTo:     targetServicePeriod.To,
		NewFullServicePeriodTo: target.FullServicePeriod.To,
		NewBillingPeriodTo:     target.BillingPeriod.To,
		NewInvoiceAt:           target.GetInvoiceAt(),
	})
	if err != nil {
		return err
	}

	return c.addPatch(existing.ID().ID, patch)
}

func (c *usageBasedChargeCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	existingCharge, ok := existing.(persistedstate.UsageBasedChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a usage based charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	if existingCharge.GetUsageBasedCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newUsageBasedChargeIntent(target)
		if err != nil {
			return err
		}

		return c.addEmulatedReplacement(existing, intent)
	}

	targetServicePeriod := target.GetServicePeriod()

	patch, err := chargesmeta.NewPatchExtend(chargesmeta.NewPatchExtendInput{
		NewServicePeriodTo:     targetServicePeriod.To,
		NewFullServicePeriodTo: target.FullServicePeriod.To,
		NewBillingPeriodTo:     target.BillingPeriod.To,
		NewInvoiceAt:           target.GetInvoiceAt(),
	})
	if err != nil {
		return err
	}

	return c.addPatch(existing.ID().ID, patch)
}

func newUsageBasedChargeIntent(target targetstate.StateItem) (charges.ChargeIntent, error) {
	rateCardMeta := target.Spec.RateCard.AsMeta()
	price := rateCardMeta.Price
	if price == nil {
		return charges.ChargeIntent{}, fmt.Errorf("price is required for usage based charge")
	}

	baseIntent, err := newChargeIntentBaseFromTargetState(target)
	if err != nil {
		return charges.ChargeIntent{}, err
	}

	intent := charges.NewChargeIntent(chargesusagebased.Intent{
		Intent:         baseIntent,
		InvoiceAt:      target.GetInvoiceAt(),
		SettlementMode: target.Subscription.SettlementMode,
		FeatureKey:     lo.FromPtr(rateCardMeta.FeatureKey),
		Price:          *price,
		Discounts:      rateCardMeta.Discounts,
	})

	return intent, nil
}
