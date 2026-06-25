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
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	if existingCharge.GetUsageBasedCharge().Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newUsageBasedChargeIntent(target)
		if err != nil {
			return err
		}

		return c.addEmulatedReplacement(existing, intent)
	}

	targetServicePeriod := target.GetServicePeriod()

	// TODO: include the charge intent target on shrink patches so subscription
	// sync mutates the base intent layer without touching API overrides.
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

	if existingCharge.GetUsageBasedCharge().Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newUsageBasedChargeIntent(target)
		if err != nil {
			return err
		}

		return c.addEmulatedReplacement(existing, intent)
	}

	targetServicePeriod := target.GetServicePeriod()

	// TODO: include the charge intent target on extend patches so subscription
	// sync mutates the base intent layer without touching API overrides.
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

	annotations, err := target.SubscriptionItem.Annotations.Clone()
	if err != nil {
		return charges.ChargeIntent{}, err
	}

	return charges.NewChargeIntent(chargesusagebased.Intent{
		Intent: chargesmeta.Intent{
			ManagedBy:         billing.SubscriptionManagedLine,
			CustomerID:        target.Subscription.CustomerId,
			Annotations:       annotations,
			Currency:          target.CurrencyCalculator.Currency,
			UniqueReferenceID: &target.UniqueID,
			Subscription: &chargesmeta.SubscriptionReference{
				SubscriptionID: target.Subscription.ID,
				PhaseID:        target.PhaseID,
				ItemID:         target.SubscriptionItem.ID,
			},
		},
		IntentMutableFields: chargesusagebased.IntentMutableFields{
			IntentMutableFields: chargesmeta.IntentMutableFields{
				Name:          rateCardMeta.Name,
				Description:   rateCardMeta.Description,
				Metadata:      target.SubscriptionItem.Metadata.Clone(),
				ServicePeriod: target.GetServicePeriod(),
				FullServicePeriod: timeutil.ClosedPeriod{
					From: target.FullServicePeriod.From,
					To:   target.FullServicePeriod.To,
				},
				BillingPeriod: timeutil.ClosedPeriod{
					From: target.BillingPeriod.From,
					To:   target.BillingPeriod.To,
				},
				TaxConfig: productcatalog.TaxCodeConfigFrom(rateCardMeta.TaxConfig),
			},
			InvoiceAt:  target.GetInvoiceAt(),
			FeatureKey: lo.FromPtr(rateCardMeta.FeatureKey),
			Price:      *price,
			Discounts:  rateCardMeta.Discounts,
		},
		SettlementMode: target.Subscription.SettlementMode,
	}), nil
}
