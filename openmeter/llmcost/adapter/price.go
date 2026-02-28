package adapter

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	pricedb "github.com/openmeterio/openmeter/openmeter/ent/db/llmcostprice"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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

		if input.Provider != nil {
			query = query.Where(pricedb.ProviderContainsFold(string(*input.Provider)))
		}

		if input.ModelID != nil {
			query = query.Where(pricedb.ModelIDEQ(*input.ModelID))
		}

		if input.ModelName != nil {
			query = query.Where(pricedb.ModelNameContainsFold(*input.ModelName))
		}

		if input.At != nil {
			query = query.
				Where(pricedb.EffectiveFromLTE(*input.At)).
				Where(
					pricedb.Or(
						pricedb.EffectiveToIsNil(),
						pricedb.EffectiveToGT(*input.At),
					),
				)
		}

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
			if db.IsNotFound(err) {
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

	// Try namespace override first
	entity, err := a.db.LLMCostPrice.Query().
		Where(pricedb.DeletedAtIsNil()).
		Where(pricedb.NamespaceEQ(input.Namespace)).
		Where(pricedb.ProviderEQ(string(input.Provider))).
		Where(pricedb.ModelIDEQ(input.ModelID)).
		Where(pricedb.EffectiveFromLTE(at)).
		Where(
			pricedb.Or(
				pricedb.EffectiveToIsNil(),
				pricedb.EffectiveToGT(at),
			),
		).
		Order(pricedb.ByEffectiveFrom(sql.OrderDesc())).
		First(ctx)
	if err == nil {
		return mapPriceFromEntity(entity)
	}

	if !db.IsNotFound(err) {
		return llmcost.Price{}, fmt.Errorf("failed to resolve price: %w", err)
	}

	// Fall back to global price
	entity, err = a.db.LLMCostPrice.Query().
		Where(pricedb.DeletedAtIsNil()).
		Where(pricedb.NamespaceIsNil()).
		Where(pricedb.ProviderEQ(string(input.Provider))).
		Where(pricedb.ModelIDEQ(input.ModelID)).
		Where(pricedb.EffectiveFromLTE(at)).
		Where(
			pricedb.Or(
				pricedb.EffectiveToIsNil(),
				pricedb.EffectiveToGT(at),
			),
		).
		Order(pricedb.ByEffectiveFrom(sql.OrderDesc())).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return llmcost.Price{}, llmcost.NewPriceNotFoundError(
				fmt.Sprintf("%s/%s", input.Provider, input.ModelID),
			)
		}

		return llmcost.Price{}, fmt.Errorf("failed to resolve global price: %w", err)
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
			SetInputCachedPerToken(decimalOrZero(input.Pricing.InputCachedPerToken)).
			SetReasoningPerToken(decimalOrZero(input.Pricing.ReasoningPerToken)).
			SetCacheWritePerToken(decimalOrZero(input.Pricing.CacheWritePerToken)).
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

// UpdateOverride updates an existing per-namespace price override.
func (a *adapter) UpdateOverride(ctx context.Context, input llmcost.UpdateOverrideInput) (llmcost.Price, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (llmcost.Price, error) {
		if err := input.Validate(); err != nil {
			return llmcost.Price{}, err
		}

		entity, err := a.db.LLMCostPrice.UpdateOneID(input.ID).
			Where(pricedb.DeletedAtIsNil()).
			Where(pricedb.NamespaceEQ(input.Namespace)).
			Where(pricedb.SourceEQ(string(llmcost.PriceSourceManual))).
			SetInputPerToken(input.Pricing.InputPerToken).
			SetOutputPerToken(input.Pricing.OutputPerToken).
			SetInputCachedPerToken(decimalOrZero(input.Pricing.InputCachedPerToken)).
			SetReasoningPerToken(decimalOrZero(input.Pricing.ReasoningPerToken)).
			SetCacheWritePerToken(decimalOrZero(input.Pricing.CacheWritePerToken)).
			SetNillableEffectiveTo(input.EffectiveTo).
			Save(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return llmcost.Price{}, llmcost.NewPriceNotFoundError(input.ID)
			}

			return llmcost.Price{}, fmt.Errorf("failed to update override: %w", err)
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
			if db.IsNotFound(err) {
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

		if input.Provider != nil {
			query = query.Where(pricedb.ProviderContainsFold(string(*input.Provider)))
		}

		if input.ModelID != nil {
			query = query.Where(pricedb.ModelIDEQ(*input.ModelID))
		}

		if input.ModelName != nil {
			query = query.Where(pricedb.ModelNameContainsFold(*input.ModelName))
		}

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
		if err != nil && !db.IsNotFound(err) {
			return fmt.Errorf("failed to query existing global price: %w", err)
		}

		if existing != nil {
			// Update existing row in place
			_, err = a.db.LLMCostPrice.UpdateOneID(existing.ID).
				SetModelName(price.ModelName).
				SetInputPerToken(price.Pricing.InputPerToken).
				SetOutputPerToken(price.Pricing.OutputPerToken).
				SetInputCachedPerToken(decimalOrZero(price.Pricing.InputCachedPerToken)).
				SetReasoningPerToken(decimalOrZero(price.Pricing.ReasoningPerToken)).
				SetCacheWritePerToken(decimalOrZero(price.Pricing.CacheWritePerToken)).
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
			SetInputCachedPerToken(decimalOrZero(price.Pricing.InputCachedPerToken)).
			SetReasoningPerToken(decimalOrZero(price.Pricing.ReasoningPerToken)).
			SetCacheWritePerToken(decimalOrZero(price.Pricing.CacheWritePerToken)).
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
