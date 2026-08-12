package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

// validateSpecFeatures applies product-catalog feature constraints to every
// subscription item, including feature-only items that do not schedule an
// entitlement.
func (s *service) validateSpecFeatures(
	ctx context.Context,
	namespace string,
	spec subscription.SubscriptionSpec,
	options ...productcatalog.ValidateRateCardsWithFeaturesOptions,
) error {
	rateCards := make(productcatalog.RateCards, 0)
	for _, phase := range spec.Phases {
		if phase == nil {
			continue
		}

		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item == nil || item.RateCard == nil {
					continue
				}

				rateCards = append(rateCards, item.RateCard)
			}
		}
	}

	return productcatalog.ValidateRateCardsWithFeatures(
		ctx,
		s.featureResolver.WithNamespace(namespace),
		options...,
	)(rateCards)
}
