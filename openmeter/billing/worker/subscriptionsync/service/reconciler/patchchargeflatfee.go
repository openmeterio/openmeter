package reconciler

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesflatfee "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/chargeupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

	return c.addPatch(target.UniqueID, PatchOperationCreate, chargeupdater.NewCreatePatch(intent))
}

func (c *flatFeeChargeCollection) AddDelete(uniqueID string, existing persistedstate.Item) error {
	return c.unsupportedOperationError(PatchOperationDelete, uniqueID, existing)
}

func (c *flatFeeChargeCollection) AddShrink(uniqueID string, existing persistedstate.Item, target targetstate.StateItem) error {
	return c.unsupportedOperationError(PatchOperationShrink, uniqueID, existing)
}

func (c *flatFeeChargeCollection) AddExtend(existing persistedstate.Item, target targetstate.StateItem) error {
	return c.unsupportedOperationError(PatchOperationExtend, target.UniqueID, existing)
}

func (c *flatFeeChargeCollection) AddProrate(existing persistedstate.Item, target targetstate.StateItem, originalPeriod, targetPeriod timeutil.ClosedPeriod, originalAmount, targetAmount alpacadecimal.Decimal) error {
	return c.unsupportedOperationError(PatchOperationProrate, target.UniqueID, existing)
}
