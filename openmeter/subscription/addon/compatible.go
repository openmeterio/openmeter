package subscriptionaddon

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// TODO: this will be moved to different package
func validateRateCards(source, target productcatalog.RateCard) error {
	if source.Key() != target.Key() {
		return fmt.Errorf("target and addon rate card keys do not match")
	}

	tMeta := target.AsMeta()
	sMeta := source.AsMeta()

	if tMeta.Price != nil && sMeta.Price != nil {
		if tMeta.Price.Type() != sMeta.Price.Type() {
			return fmt.Errorf("target and addon rate card price types do not match")
		}

		if tMeta.Price.Type() == productcatalog.FlatPriceType {
			tFlat, _ := tMeta.Price.AsFlat()
			sFlat, _ := sMeta.Price.AsFlat()

			if tFlat.PaymentTerm != sFlat.PaymentTerm {
				return fmt.Errorf("target and addon rate card price payment terms do not match")
			}
		}
	}

	if tMeta.EntitlementTemplate != nil && sMeta.EntitlementTemplate != nil && tMeta.EntitlementTemplate.Type() != sMeta.EntitlementTemplate.Type() {
		return fmt.Errorf("target and addon rate card entitlement types do not match")
	}

	return nil
}
