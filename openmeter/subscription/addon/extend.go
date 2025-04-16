package subscriptionaddon

import (
	"fmt"
	"reflect"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Apply applies the addon rate card to the target rate card
func (a SubscriptionAddonRateCard) Apply(target productcatalog.RateCard) error {
	// Target has has to be implemented by a pointer otherwise we can't use it as a receiver. Let's check that
	typ := reflect.TypeOf(target)
	if typ == nil {
		return fmt.Errorf("target must not be nil")
	}

	if typ.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	if a.AddonRateCard.AsMeta().Price == nil && a.AddonRateCard.AsMeta().EntitlementTemplate == nil {
		return nil
	}

	if err := validateRateCards(a.AddonRateCard.RateCard, target); err != nil {
		return err
	}

	return target.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		aMeta := a.AddonRateCard.AsMeta()
		tMeta := m.Clone()

		// Let's update the price
		if aMeta.Price != nil {
			switch {
			case tMeta.Price == nil:
				m.Price = aMeta.Price
			case tMeta.Price.Type() == productcatalog.FlatPriceType:
				tFlat, _ := tMeta.Price.AsFlat()
				aFlat, _ := aMeta.Price.AsFlat()

				m.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      tFlat.Amount.Add(aFlat.Amount),
					PaymentTerm: tFlat.PaymentTerm,
				})
			default:
				return m, fmt.Errorf("not supported price type: %s", tMeta.Price.Type())
			}
		}

		// Let's update the entitlement template
		if aMeta.EntitlementTemplate != nil {
			switch {
			case tMeta.EntitlementTemplate == nil:
				m.EntitlementTemplate = aMeta.EntitlementTemplate
			case tMeta.EntitlementTemplate.Type().String() == entitlement.EntitlementTypeBoolean.String():
				// no-op
			case tMeta.EntitlementTemplate.Type().String() == entitlement.EntitlementTypeMetered.String():
				tMetered, _ := tMeta.EntitlementTemplate.AsMetered()
				aMetered, _ := aMeta.EntitlementTemplate.AsMetered()

				tMetered.IssueAfterReset = lo.ToPtr(lo.FromPtrOr(tMetered.IssueAfterReset, 0) + lo.FromPtrOr(aMetered.IssueAfterReset, 0))
				m.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(tMetered)
			default:
				return m, fmt.Errorf("not supported entitlement template type: %s", tMeta.EntitlementTemplate.Type())
			}
		}

		return m, nil
	})
}

// Restore restores the addon rate card to the target rate card
func (a SubscriptionAddonRateCard) Restore(target productcatalog.RateCard) error {
	// TODO: implement
	return models.NewGenericNotImplementedError(fmt.Errorf("not implemented"))
}
