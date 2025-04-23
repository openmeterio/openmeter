package subscriptionaddon

import (
	"errors"
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
				return m, fmt.Errorf("target price is nil, cannot restore price without addon")
			case tMeta.Price.Type() == productcatalog.FlatPriceType:
				tFlat, _ := tMeta.Price.AsFlat()
				aFlat, _ := aMeta.Price.AsFlat()

				newAmount := tFlat.Amount.Sub(aFlat.Amount)
				if newAmount.IsNegative() {
					return m, fmt.Errorf("restoring flat price would yield a negative amount: %s - %s = %s", tFlat.Amount, aFlat.Amount, newAmount)
				}

				m.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      newAmount,
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
				return m, fmt.Errorf("target entitlement template is nil, cannot restore entitlement template without addon")
			case tMeta.EntitlementTemplate.Type().String() == entitlement.EntitlementTypeBoolean.String():
				// TODO: figure this out, we have a data loss situation here
				return m, models.NewGenericNotImplementedError(errors.New("boolean entitlement templates are not supported"))
			case tMeta.EntitlementTemplate.Type().String() == entitlement.EntitlementTypeMetered.String():
				tMetered, _ := tMeta.EntitlementTemplate.AsMetered()
				aMetered, _ := aMeta.EntitlementTemplate.AsMetered()

				newIssueAfterReset := lo.FromPtrOr(tMetered.IssueAfterReset, 0) - lo.FromPtrOr(aMetered.IssueAfterReset, 0)

				if newIssueAfterReset < 0 {
					return m, fmt.Errorf("restoring entitlement template would yield a negative issue after reset: %.0f - %.0f = %.0f", lo.FromPtrOr(tMetered.IssueAfterReset, 0), lo.FromPtrOr(aMetered.IssueAfterReset, 0), newIssueAfterReset)
				}

				tMetered.IssueAfterReset = lo.ToPtr(newIssueAfterReset)
				m.EntitlementTemplate = productcatalog.NewEntitlementTemplateFrom(tMetered)
			default:
				return m, fmt.Errorf("not supported entitlement template type: %s", tMeta.EntitlementTemplate.Type())
			}
		}

		return m, nil
	})
}
