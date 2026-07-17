package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type costBasisPair struct {
	customCurrencyID string
	invoiceCurrency  string
}

func customCostBasisPairs(spec subscription.SubscriptionSpec) map[costBasisPair]struct{} {
	pairs := map[costBasisPair]struct{}{}

	for _, phase := range spec.Phases {
		if phase == nil {
			continue
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

				managed, ok := meta.Currency.(currencyx.ManagedCurrency)
				if !ok || managed.GetID() == "" {
					continue
				}

				pairs[costBasisPair{
					customCurrencyID: managed.GetID(),
					invoiceCurrency:  spec.InvoiceCurrency.String(),
				}] = struct{}{}
			}
		}
	}

	return pairs
}

func (s *service) resolveCostBasisRequirements(
	ctx context.Context,
	namespace string,
	spec subscription.SubscriptionSpec,
	at time.Time,
	pairs map[costBasisPair]struct{},
) ([]currencies.CostBasis, error) {
	if spec.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode || len(pairs) == 0 {
		return nil, nil
	}

	orderedPairs := make([]costBasisPair, 0, len(pairs))
	for pair := range pairs {
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
		costBasis, err := s.CostBasisService.GetCostBasisAt(ctx, currencies.GetCostBasisAtInput{
			Namespace:  namespace,
			CurrencyID: pair.customCurrencyID,
			FiatCode:   spec.InvoiceCurrency,
			At:         at,
		})
		if err != nil {
			if models.IsGenericNotFoundError(err) {
				return nil, models.NewGenericValidationError(fmt.Errorf(
					"%w [custom_currency_id=%s invoice_currency=%s at=%s]",
					productcatalog.ErrCurrencyCostBasisNotFound,
					pair.customCurrencyID,
					spec.InvoiceCurrency,
					at.UTC().Format(time.RFC3339),
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
			InvoiceCurrency:  currencyx.Code(costBasis.FiatCode),
			CostBasisID:      costBasis.ID,
		})
	}

	return inputs
}
