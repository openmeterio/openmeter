package reconciler

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type flatFeeChargeCollection struct {
	chargePatchCollection
}

func newFlatFeeChargeCollection(preallocatedCapacity int) *flatFeeChargeCollection {
	return &flatFeeChargeCollection{
		chargePatchCollection: newChargePatchCollection(persistedstate.ItemTypeChargeFlatFee, preallocatedCapacity),
	}
}

func (c *flatFeeChargeCollection) AddCreate(target targetstate.StateItem) error {
	rateCardMeta := target.Spec.RateCard.AsMeta()
	price := rateCardMeta.Price
	if price == nil {
		return fmt.Errorf("price is required for flat fee charge")
	}

	flatPrice, err := price.AsFlat()
	if err != nil {
		return fmt.Errorf("converting price to flat: %w", err)
	}

	baseIntent, err := newChargeIntentBaseFromTargetState(target)
	if err != nil {
		return err
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

	return c.addCreate(intent)
}
