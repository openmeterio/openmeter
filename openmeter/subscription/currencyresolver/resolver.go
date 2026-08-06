package currencyresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	productcatalogcurrencyresolver "github.com/openmeterio/openmeter/openmeter/productcatalog/currencyresolver"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

// ResolveCurrenciesForSubscriptionSpec resolves the materialized currency of
// every subscription item in one batch. The service must call this before
// validating or persisting a spec so repository inputs carry authoritative
// custom-currency identity.
func ResolveCurrenciesForSubscriptionSpec(
	ctx context.Context,
	resolver currencies.NamespacedCurrencyResolver,
	spec *subscription.SubscriptionSpec,
) error {
	if spec == nil {
		return errors.New("subscription spec is required")
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

	if err := productcatalogcurrencyresolver.ResolveCurrenciesForRateCards(ctx, resolver, &rateCards); err != nil {
		return fmt.Errorf("resolving subscription item currencies: %w", err)
	}

	return nil
}
