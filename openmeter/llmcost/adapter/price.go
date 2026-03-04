package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/api/v3/filters"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	pricedb "github.com/openmeterio/openmeter/openmeter/ent/db/llmcostprice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListPrices returns global (namespace IS NULL) prices with optional filtering.
func (a *adapter) ListPrices(ctx context.Context, input llmcost.ListPricesInput) (pagination.Result[llmcost.Price], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (pagination.Result[llmcost.Price], error) {
		if err := input.Validate(); err != nil {
			return pagination.Result[llmcost.Price]{}, err
		}

		query := a.db.LLMCostPrice.Query().
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.NamespaceIsNil()) // Global prices only

		applyStringFilter(input.Provider, &query,
			pricedb.ProviderEqualFold, pricedb.ProviderNEQ, pricedb.ProviderContainsFold,
		)
		applyStringFilter(input.ModelID, &query,
			pricedb.ModelIDEqualFold, pricedb.ModelIDNEQ, pricedb.ModelIDContainsFold,
		)
		applyStringFilter(input.ModelName, &query,
			pricedb.ModelNameEqualFold, pricedb.ModelNameNEQ, pricedb.ModelNameContainsFold,
		)
		applyStringFilter(input.Currency, &query,
			pricedb.CurrencyEqualFold, pricedb.CurrencyNEQ, pricedb.CurrencyContainsFold,
		)

		query = query.Order(pricedb.ByProvider(), pricedb.ByModelID(), pricedb.ByEffectiveFrom())

		entities, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[llmcost.Price]{}, fmt.Errorf("failed to list prices: %w", err)
		}

		return pagination.MapResultErr(entities, mapPriceFromEntity)
	})
}

// GetPrice returns a specific price by ID.
func (a *adapter) GetPrice(ctx context.Context, input llmcost.GetPriceInput) (llmcost.Price, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) {
		if err := input.Validate(); err != nil {
			return llmcost.Price{}, err
		}

		entity, err := a.db.LLMCostPrice.Query().
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.ID(input.ID)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return llmcost.Price{}, llmcost.NewPriceNotFoundError(input.ID)
			}

			return llmcost.Price{}, fmt.Errorf("failed to get price: %w", err)
		}

		return mapPriceFromEntity(entity)
	})
}

// ResolvePrice finds the effective price, preferring namespace override > global.
// This is a read-only operation and does not use a transaction.
func (a *adapter) ResolvePrice(ctx context.Context, input llmcost.ResolvePriceInput) (llmcost.Price, error) {
	if err := input.Validate(); err != nil {
		return llmcost.Price{}, err
	}

	at := clock.Now()
	if input.At != nil {
		at = *input.At
	}

	entity, err := a.db.LLMCostPrice.Query().
		Where(pricedb.DeletedAtIsNil()).
		Where(
			pricedb.Or(
				// If namespace is set, try to find an override
				pricedb.NamespaceEQ(input.Namespace),
				// If namespace is not set, try to find a global price
				pricedb.NamespaceIsNil(),
			),
		).
		Where(pricedb.ProviderEQ(string(input.Provider))).
		Where(pricedb.ModelIDEQ(input.ModelID)).
		Where(pricedb.EffectiveFromLTE(at)).
		Where(
			pricedb.Or(
				pricedb.EffectiveToIsNil(),
				pricedb.EffectiveToGT(at),
			),
		).
		Order(
			// Prioritize namespace overrides
			pricedb.ByNamespace(sql.OrderDesc()),
			// Then effective from
			pricedb.ByEffectiveFrom(sql.OrderDesc()),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return llmcost.Price{}, models.NewGenericNotFoundError(llmcost.NewPriceNotFoundError(input.ModelID))
		}

		return llmcost.Price{}, fmt.Errorf("failed to resolve price: %w", err)
	}

	return mapPriceFromEntity(entity)
}

// CreateOverride creates a per-namespace price override.
func (a *adapter) CreateOverride(ctx context.Context, input llmcost.CreateOverrideInput) (llmcost.Price, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) {
		if err := input.Validate(); err != nil {
			return llmcost.Price{}, err
		}

		// Soft-delete any existing active override for the same provider/model/namespace
		_, err := a.db.LLMCostPrice.Update().
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.NamespaceEQ(input.Namespace)).
			Where(pricedb.ProviderEQ(string(input.Provider))).
			Where(pricedb.ModelIDEQ(input.ModelID)).
			Where(pricedb.SourceEQ(string(llmcost.PriceSourceManual))).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if err != nil {
			return llmcost.Price{}, fmt.Errorf("failed to soft-delete previous override: %w", err)
		}

		entity, err := a.db.LLMCostPrice.Create().
			SetNamespace(input.Namespace).
			SetProvider(string(input.Provider)).
			SetModelID(input.ModelID).
			SetModelName(input.ModelName).
			SetInputPerToken(input.Pricing.InputPerToken).
			SetOutputPerToken(input.Pricing.OutputPerToken).
			SetCacheReadPerToken(decimalOrZero(input.Pricing.CacheReadPerToken)).
			SetCacheWritePerToken(decimalOrZero(input.Pricing.CacheWritePerToken)).
			SetReasoningPerToken(decimalOrZero(input.Pricing.ReasoningPerToken)).
			SetCurrency(input.Currency).
			SetSource(string(llmcost.PriceSourceManual)).
			SetEffectiveFrom(input.EffectiveFrom).
			SetNillableEffectiveTo(input.EffectiveTo).
			Save(ctx)
		if err != nil {
			return llmcost.Price{}, fmt.Errorf("failed to create override: %w", err)
		}

		return mapPriceFromEntity(entity)
	})
}

// DeleteOverride soft-deletes a per-namespace price override.
func (a *adapter) DeleteOverride(ctx context.Context, input llmcost.DeleteOverrideInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, a *adapter) error {
		if err := input.Validate(); err != nil {
			return err
		}

		entity, err := a.db.LLMCostPrice.Query().
			Where(pricedb.ID(input.ID)).
			Where(pricedb.NamespaceEQ(input.Namespace)).
			Where(pricedb.SourceEQ(string(llmcost.PriceSourceManual))).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return llmcost.NewPriceNotFoundError(input.ID)
			}

			return fmt.Errorf("failed to get override: %w", err)
		}

		if entity.DeletedAt == nil {
			err := a.db.LLMCostPrice.UpdateOneID(input.ID).
				Where(pricedb.NamespaceEQ(input.Namespace)).
				SetDeletedAt(clock.Now()).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to delete override: %w", err)
			}
		}

		return nil
	})
}

// ListOverrides returns per-namespace overrides.
func (a *adapter) ListOverrides(ctx context.Context, input llmcost.ListOverridesInput) (pagination.Result[llmcost.Price], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (pagination.Result[llmcost.Price], error) {
		if err := input.Validate(); err != nil {
			return pagination.Result[llmcost.Price]{}, err
		}

		query := a.db.LLMCostPrice.Query().
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.NamespaceEQ(input.Namespace)).
			Where(pricedb.SourceEQ(string(llmcost.PriceSourceManual)))

		applyStringFilter(input.Provider, &query,
			pricedb.ProviderEqualFold, pricedb.ProviderNEQ, pricedb.ProviderContainsFold,
		)
		applyStringFilter(input.ModelID, &query,
			pricedb.ModelIDEqualFold, pricedb.ModelIDNEQ, pricedb.ModelIDContainsFold,
		)
		applyStringFilter(input.ModelName, &query,
			pricedb.ModelNameEqualFold, pricedb.ModelNameNEQ, pricedb.ModelNameContainsFold,
		)
		applyStringFilter(input.Currency, &query,
			pricedb.CurrencyEqualFold, pricedb.CurrencyNEQ, pricedb.CurrencyContainsFold,
		)

		query = query.Order(pricedb.ByProvider(), pricedb.ByModelID())

		entities, err := query.Paginate(ctx, input.Page)
		if err != nil {
			return pagination.Result[llmcost.Price]{}, fmt.Errorf("failed to list overrides: %w", err)
		}

		return pagination.MapResultErr(entities, mapPriceFromEntity)
	})
}

// UpsertGlobalPrice creates or updates the current global price for a provider+model.
func (a *adapter) UpsertGlobalPrice(ctx context.Context, price llmcost.Price) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, a *adapter) error {
		// Try to find existing current global price
		existing, err := a.db.LLMCostPrice.Query().
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.NamespaceIsNil()).
			Where(pricedb.ProviderEQ(string(price.Provider))).
			Where(pricedb.ModelIDEQ(price.ModelID)).
			Where(pricedb.EffectiveToIsNil()).
			First(ctx)
		if err != nil && !entdb.IsNotFound(err) {
			return fmt.Errorf("failed to query existing global price: %w", err)
		}

		if existing != nil {
			// Update existing row in place
			_, err = a.db.LLMCostPrice.UpdateOneID(existing.ID).
				SetModelName(price.ModelName).
				SetInputPerToken(price.Pricing.InputPerToken).
				SetOutputPerToken(price.Pricing.OutputPerToken).
				SetCacheReadPerToken(decimalOrZero(price.Pricing.CacheReadPerToken)).
				SetCacheWritePerToken(decimalOrZero(price.Pricing.CacheWritePerToken)).
				SetReasoningPerToken(decimalOrZero(price.Pricing.ReasoningPerToken)).
				SetCurrency(price.Currency).
				SetSource(string(price.Source)).
				SetSourcePrices(price.SourcePrices).
				Save(ctx)
			if err != nil {
				return fmt.Errorf("failed to update global price: %w", err)
			}

			return nil
		}

		// Create new row
		err = a.db.LLMCostPrice.Create().
			SetProvider(string(price.Provider)).
			SetModelID(price.ModelID).
			SetModelName(price.ModelName).
			SetInputPerToken(price.Pricing.InputPerToken).
			SetOutputPerToken(price.Pricing.OutputPerToken).
			SetCacheReadPerToken(decimalOrZero(price.Pricing.CacheReadPerToken)).
			SetCacheWritePerToken(decimalOrZero(price.Pricing.CacheWritePerToken)).
			SetReasoningPerToken(decimalOrZero(price.Pricing.ReasoningPerToken)).
			SetCurrency(price.Currency).
			SetSource(string(price.Source)).
			SetSourcePrices(price.SourcePrices).
			SetEffectiveFrom(price.EffectiveFrom).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create global price: %w", err)
		}

		return nil
	})
}

// applyStringFilter applies a StringFilter to an ent query using the provided predicate functions.
func applyStringFilter(
	f *filters.StringFilter,
	query **entdb.LLMCostPriceQuery,
	eqFold func(string) predicate.LLMCostPrice,
	neq func(string) predicate.LLMCostPrice,
	containsFold func(string) predicate.LLMCostPrice,
) {
	if f == nil {
		return
	}

	switch {
	case f.Eq != nil:
		*query = (*query).Where(eqFold(*f.Eq))
	case f.Neq != nil:
		*query = (*query).Where(neq(*f.Neq))
	case f.Contains != nil:
		*query = (*query).Where(containsFold(*f.Contains))
	}
}
