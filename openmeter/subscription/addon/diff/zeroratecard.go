package addondiff

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

// A zeroRateCardCheck is a check that determines if a RateCard can be deleted
// A RateCard can be deleted if it's effectively zero (as determined by contents and item annotations)
type zeroRateCardCheck struct {
	rc              productcatalog.RateCard
	itemAnnotations models.Annotations
}

func (z zeroRateCardCheck) CanDelete() bool {
	return z.isZeroByContents() && z.isZeroByAnnotations()
}

// For contents, we only measure fields on which subscriptionaddon.RateCard.Apply/Restore operates
// These are:
// - Price
// - EntitlementTemplate
func (z zeroRateCardCheck) isZeroByContents() bool {
	m := z.rc.AsMeta()

	// We check for non-0 flat price or any non-flat price
	if m.Price != nil {
		switch m.Price.Type() {
		case productcatalog.FlatPriceType:
			f, err := m.Price.AsFlat()
			if err != nil {
				return false
			}

			if !f.Amount.IsZero() {
				return false
			}

		default:
			return false
		}
	}

	// We check for non-0 entitlement template
	if m.EntitlementTemplate != nil {
		switch m.EntitlementTemplate.Type() {
		case entitlement.EntitlementTypeMetered:
			mt, err := m.EntitlementTemplate.AsMetered()
			if err != nil {
				return false
			}

			if mt.IssueAfterReset != nil && *mt.IssueAfterReset != 0 {
				return false
			}
		case entitlement.EntitlementTypeBoolean:
		default:
			return false
		}
	}

	return true
}

// For annotations, we check whether the item inherits from the subscription systems owner annotation
func (z zeroRateCardCheck) isZeroByAnnotations() bool {
	ownerSystems := subscription.AnnotationParser.ListOwnerSubSystems(z.itemAnnotations)

	return !lo.Contains(ownerSystems, subscription.OwnerSubscriptionSubSystem)
}
