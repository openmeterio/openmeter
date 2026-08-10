package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogfeatureresolver "github.com/openmeterio/openmeter/openmeter/productcatalog/featureresolver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

// prepareSpecFeatures resolves every subscription item feature before a create
// or update so feature-only items receive the same validation as items that
// schedule entitlements.
func (s *service) prepareSpecFeatures(ctx context.Context, namespace string, spec *subscription.SubscriptionSpec) error {
	if spec == nil {
		return fmt.Errorf("subscription spec is required")
	}

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

	if err := productcatalogfeatureresolver.ResolveFeaturesForRateCards(
		ctx,
		s.featureResolver,
		namespace,
		&rateCards,
	); err != nil {
		return fmt.Errorf("resolving subscription item features: %w", err)
	}

	return nil
}
