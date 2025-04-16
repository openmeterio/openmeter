package subscriptionaddon

import (
	"fmt"
	"reflect"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// Apply applies the addon rate card to the target rate card
func (a SubscriptionAddonRateCard) Apply(target productcatalog.RateCard) error {
	// Target has has to be implemented by a pointer otherwise we can't use it as a receiver. Let's check that
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	if target.AsMeta().Price.Type() != a.AddonRateCard.AsMeta().Price.Type() {
		return fmt.Errorf("target and addon rate card price types do not match")
	}

	// TODO: implement

	return target.ChangeMeta(func(m productcatalog.RateCardMeta) productcatalog.RateCardMeta {
		switch m.Price.Type() {
		case productcatalog.FlatPriceType:
			flat, _ := m.Price.AsFlat()

			aFlat, _ := a.AddonRateCard.RateCard.AsMeta().Price.AsFlat()

			m.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      flat.Amount.Add(aFlat.Amount),
				PaymentTerm: flat.PaymentTerm,
			})
		default:
			panic("not implemented")
		}

		return m
	})
}

// Restore restores the addon rate card to the target rate card
func (a SubscriptionAddonRateCard) Restore(target productcatalog.RateCard) error {
	// TODO: implement
	return nil
}
