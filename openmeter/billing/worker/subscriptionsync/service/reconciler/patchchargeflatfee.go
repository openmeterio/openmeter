package reconciler

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type flatFeeChargeCollection struct {
	chargePatchCollection
}

func newFlatFeeChargeCollection(preallocatedCapacity int) *flatFeeChargeCollection {
	return &flatFeeChargeCollection{
		chargePatchCollection: newChargePatchCollection(billing.LineEngineTypeChargeFlatFee, persistedstate.ItemTypeChargeFlatFee, preallocatedCapacity),
	}
}

func (c *flatFeeChargeCollection) AddCreate(target targetstate.StateItem) error {
	intent, err := newFlatFeeChargeIntent(target)
	if err != nil {
		return err
	}

	return c.addCreate(intent)
}

func (c *flatFeeChargeCollection) AddShrink(_ string, existing persistedstate.Item, target targetstate.StateItem) error {
	existingCharge, ok := existing.(persistedstate.FlatFeeChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a flat fee charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	if existingCharge.GetFlatFeeCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newFlatFeeChargeIntent(target)
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

func (c *flatFeeChargeCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	existingCharge, ok := existing.(persistedstate.FlatFeeChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a flat fee charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	if existingCharge.GetFlatFeeCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		intent, err := newFlatFeeChargeIntent(target)
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

func newFlatFeeChargeIntent(target targetstate.StateItem) (charges.ChargeIntent, error) {
	rateCardMeta := target.Spec.RateCard.AsMeta()
	price := rateCardMeta.Price
	if price == nil {
		return charges.ChargeIntent{}, fmt.Errorf("price is required for flat fee charge")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return charges.ChargeIntent{}, fmt.Errorf("converting price to flat: %w", err)
	}

	baseIntent, err := newChargeIntentBaseFromTargetState(target)
	if err != nil {
		return charges.ChargeIntent{}, err
	}

	intent := charges.NewChargeIntent(chargesflatfee.Intent{
		Intent:                baseIntent,
		InvoiceAt:             target.GetInvoiceAt(),
		SettlementMode:        target.Subscription.SettlementMode,
		PaymentTerm:           flatPrice.PaymentTerm,
		FeatureKey:            lo.FromPtr(rateCardMeta.FeatureKey),
		PercentageDiscounts:   rateCardMeta.Discounts.Percentage,
		ProRating:             target.Subscription.ProRatingConfig,
		AmountBeforeProration: flatPrice.Amount,
	})

	return intent, nil
}
