package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptioncurrencyresolver "github.com/openmeterio/openmeter/openmeter/subscription/currencyresolver"
	"github.com/openmeterio/openmeter/pkg/models"
)

// prepareSpecCurrencies snapshots inherited item currencies and resolves every
// custom reference before validation and persistence.
func (s *service) prepareSpecCurrencies(ctx context.Context, namespace string, spec *subscription.SubscriptionSpec) error {
	if spec == nil {
		return fmt.Errorf("subscription spec is required")
	}

	if err := spec.MaterializeRateCardCurrencies(currencies.NewCurrencyReference(spec.InvoiceCurrency)); err != nil {
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

type costBasisPair struct {
	customCurrencyID string
	invoiceCurrency  string
}

func customCostBasisPairs(spec subscription.SubscriptionSpec) map[costBasisPair]time.Time {
	pairs := map[costBasisPair]time.Time{}

	for _, phase := range spec.Phases {
		if phase == nil {
			continue
		}

		initialActiveFrom := spec.ActiveFrom
		if !spec.CostBasisMode.IsPinned() {
			phaseStart, _ := phase.StartAfter.AddTo(spec.ActiveFrom)
			initialActiveFrom = lo.Latest(phaseStart, lo.FromPtr(spec.BillingStartsAt)).Truncate(streaming.MinimumWindowSizeDuration)
		}

		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if item == nil || item.RateCard == nil {
					continue
				}

				meta := item.RateCard.AsMeta()
				if meta.Price == nil || meta.Currency == nil || !meta.Currency.IsCustom() {
					continue
				}

				if meta.Currency.CustomCurrencyID == nil || *meta.Currency.CustomCurrencyID == "" {
					continue
				}

				pair := costBasisPair{
					customCurrencyID: *meta.Currency.CustomCurrencyID,
					invoiceCurrency:  spec.InvoiceCurrency.String(),
				}
				if first, exists := pairs[pair]; !exists || initialActiveFrom.Before(first) {
					pairs[pair] = initialActiveFrom
				}
			}
		}
	}

	return pairs
}

func (s *service) resolveCostBasisRequirements(
	ctx context.Context,
	namespace string,
	spec subscription.SubscriptionSpec,
	lookupTimes map[costBasisPair]time.Time,
) ([]currencies.CostBasis, error) {
	if spec.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode || len(lookupTimes) == 0 {
		return nil, nil
	}

	orderedPairs := make([]costBasisPair, 0, len(lookupTimes))
	for pair := range lookupTimes {
		orderedPairs = append(orderedPairs, pair)
	}

	slices.SortFunc(orderedPairs, func(a, b costBasisPair) int {
		if a.customCurrencyID == b.customCurrencyID {
			return strings.Compare(a.invoiceCurrency, b.invoiceCurrency)
		}

		return strings.Compare(a.customCurrencyID, b.customCurrencyID)
	})

	resolved := make([]currencies.CostBasis, 0, len(orderedPairs))
	for _, pair := range orderedPairs {
		lookupTime := lookupTimes[pair]
		costBasis, err := s.CostBasisService.GetCostBasisAt(ctx, currencies.GetCostBasisAtInput{
			Namespace:  namespace,
			CurrencyID: pair.customCurrencyID,
			FiatCode:   spec.InvoiceCurrency,
			At:         lookupTime,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				return nil, models.NewGenericValidationError(fmt.Errorf(
					"%w [custom_currency_id=%s invoice_currency=%s at=%s]",
					productcatalog.ErrCurrencyCostBasisNotFound,
					pair.customCurrencyID,
					spec.InvoiceCurrency,
					lookupTime.UTC().Format(time.RFC3339),
				))
			}

			return nil, fmt.Errorf("resolving subscription cost basis: %w", err)
		}

		resolved = append(resolved, costBasis)
	}

	return resolved, nil
}

func costBasisPinInputs(namespace, subscriptionID string, costBases []currencies.CostBasis) []subscription.CreateCostBasisPinEntityInput {
	inputs := make([]subscription.CreateCostBasisPinEntityInput, 0, len(costBases))
	for _, costBasis := range costBases {
		inputs = append(inputs, subscription.CreateCostBasisPinEntityInput{
			Namespace:        namespace,
			SubscriptionID:   subscriptionID,
			CustomCurrencyID: costBasis.CurrencyID,
			InvoiceCurrency:  costBasis.FiatCode,
			CostBasisID:      costBasis.ID,
		})
	}

	return inputs
}
