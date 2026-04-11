package service

import (
	"context"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type service struct {
	adapter llmcost.Adapter
	logger  *slog.Logger
}

func New(adapter llmcost.Adapter, logger *slog.Logger) llmcost.Service {
	return &service{
		adapter: adapter,
		logger:  logger,
	}
}

func (s *service) ListPrices(ctx context.Context, input llmcost.ListPricesInput) (pagination.Result[llmcost.Price], error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) {
		result, err := s.adapter.ListPrices(ctx, input)
		if err != nil {
			return pagination.Result[llmcost.Price]{}, err
		}

		// If no namespace, return global prices as-is
		if input.Namespace == "" || len(result.Items) == 0 {
			return result, nil
		}

		// Overrides are always source=manual. If the source filter excludes
		// manual prices, skip the overlay to avoid violating the filter.
		if sourceFilterExcludesManual(input.Source) {
			return result, nil
		}

		// Batch-fetch all overrides for this namespace in a single query
		overrides, err := s.adapter.ListOverrides(ctx, llmcost.ListOverridesInput{
			Namespace: input.Namespace,
			Provider:  input.Provider,
			ModelID:   input.ModelID,
			ModelName: input.ModelName,
			Currency:  input.Currency,
		})
		if err != nil {
			return pagination.Result[llmcost.Price]{}, err
		}

		if len(overrides.Items) == 0 {
			return result, nil
		}

		// Index overrides by provider+model_id for O(1) lookup
		type overrideKey struct {
			Provider string
			ModelID  string
		}
		overrideMap := make(map[overrideKey]llmcost.Price, len(overrides.Items))
		for _, o := range overrides.Items {
			overrideMap[overrideKey{string(o.Provider), o.ModelID}] = o
		}

		// Replace global prices with overrides where available
		for i, p := range result.Items {
			if o, ok := overrideMap[overrideKey{string(p.Provider), p.ModelID}]; ok {
				result.Items[i] = o
			}
		}

		return result, nil
	})
}

func (s *service) GetPrice(ctx context.Context, input llmcost.GetPriceInput) (llmcost.Price, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (llmcost.Price, error) {
		price, err := s.adapter.GetPrice(ctx, input)
		if err != nil {
			return llmcost.Price{}, err
		}

		// If namespace is set and the price is global, check for a namespace override
		if input.Namespace == "" || price.Namespace != nil {
			return price, nil
		}

		override, err := s.adapter.ResolvePrice(ctx, llmcost.ResolvePriceInput{
			Namespace: input.Namespace,
			Provider:  price.Provider,
			ModelID:   price.ModelID,
		})
		if err != nil {
			// No override found, return the global price
			if models.IsGenericNotFoundError(err) {
				return price, nil
			}

			return llmcost.Price{}, err
		}

		// Return the override if it's a manual (namespace) override
		if override.Source == llmcost.PriceSourceManual {
			return override, nil
		}

		return price, nil
	})
}

func (s *service) ResolvePrice(ctx context.Context, input llmcost.ResolvePriceInput) (llmcost.Price, error) {
	return s.adapter.ResolvePrice(ctx, input)
}

func (s *service) CreateOverride(ctx context.Context, input llmcost.CreateOverrideInput) (llmcost.Price, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (llmcost.Price, error) {
		return s.adapter.CreateOverride(ctx, input)
	})
}

func (s *service) DeleteOverride(ctx context.Context, input llmcost.DeleteOverrideInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteOverride(ctx, input)
	})
}

func (s *service) ListOverrides(ctx context.Context, input llmcost.ListOverridesInput) (pagination.Result[llmcost.Price], error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[llmcost.Price], error) {
		return s.adapter.ListOverrides(ctx, input)
	})
}

// sourceFilterExcludesManual returns true when the source filter would exclude
// manual prices. Since namespace overrides are always source=manual, the overlay
// must be skipped when manual is excluded to avoid violating the filter.
func sourceFilterExcludesManual(source *filter.FilterString) bool {
	if source == nil {
		return false
	}

	manual := string(llmcost.PriceSourceManual)

	if source.Eq != nil && *source.Eq != manual {
		return true
	}

	if source.Ne != nil && *source.Ne == manual {
		return true
	}

	return false
}
