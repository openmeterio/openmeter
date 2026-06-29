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
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
	_, ok := existing.(persistedstate.FlatFeeChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a flat fee charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	patch, err := chargesmeta.NewPatchShrink(chargesmeta.NewPatchShrinkInput{
		Target:                 chargesmeta.ChangeTargetBase,
		NewServicePeriodTo:     target.GetServicePeriod().To,
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
	_, ok := existing.(persistedstate.FlatFeeChargeGetter)
	if !ok {
		return fmt.Errorf("existing item is not a flat fee charge [item_type=%s,id=%s]", existing.Type(), existing.ID())
	}

	patch, err := chargesmeta.NewPatchExtend(chargesmeta.NewPatchExtendInput{
		Target:                 chargesmeta.ChangeTargetBase,
		NewServicePeriodTo:     target.GetServicePeriod().To,
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

	annotations, err := target.SubscriptionItem.Annotations.Clone()
	if err != nil {
		return charges.ChargeIntent{}, err
	}

	return charges.NewChargeIntent(chargesflatfee.Intent{
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
		IntentMutableFields: chargesflatfee.IntentMutableFields{
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
			InvoiceAt:             target.GetInvoiceAt(),
			PaymentTerm:           flatPrice.PaymentTerm,
			FeatureKey:            lo.FromPtr(rateCardMeta.FeatureKey),
			PercentageDiscounts:   billing.DiscountsFromProductCatalog(rateCardMeta.Discounts).UpsertCorrelationIDs().Percentage,
			ProRating:             target.Subscription.ProRatingConfig,
			AmountBeforeProration: flatPrice.Amount,
		},
		SettlementMode: target.Subscription.SettlementMode,
	}), nil
}
