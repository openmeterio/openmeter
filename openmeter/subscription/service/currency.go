package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptioncurrencyresolver "github.com/openmeterio/openmeter/openmeter/subscription/currencyresolver"
)

// prepareSpecCurrencies snapshots inherited item currencies and resolves every
// custom reference before validation and persistence.
func (s *service) prepareSpecCurrencies(ctx context.Context, namespace string, spec *subscription.SubscriptionSpec) error {
	if spec == nil {
		return fmt.Errorf("subscription spec is required")
	}

	if err := spec.MaterializeRateCardCurrencies(currencies.NewCurrencyReference(spec.Currency)); err != nil {
		return fmt.Errorf("failed to materialize subscription item currencies: %w", err)
	}

	if err := subscriptioncurrencyresolver.ResolveCurrenciesForSubscriptionSpec(
		ctx,
		s.CurrencyResolver.WithNamespace(namespace),
		spec,
	); err != nil {
		return fmt.Errorf("failed to resolve subscription item currencies: %w", err)
	}

	return nil
}
