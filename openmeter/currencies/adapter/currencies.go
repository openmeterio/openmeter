package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/samber/lo"
)

var _ currencies.Adapter = (*adapter)(nil)

func mapCurrencyFromDB(c *entdb.CustomCurrency) currencies.Currency {
	return currencies.Currency{
		NamespacedID: models.NamespacedID{ID: c.ID, Namespace: c.Namespace},
		ManagedModel: models.ManagedModel{CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, DeletedAt: c.DeletedAt},
		Code:         c.Code,
		Name:         c.Name,
		Symbol:       lo.ToPtr(c.Symbol),
	}
}

func mapCostBasisFromDB(c *entdb.CurrencyCostBasis) currencies.CostBasis {
	var effectiveTo *time.Time
	if c.EffectiveTo != nil {
		t := c.EffectiveTo.In(time.UTC)
		effectiveTo = &t
	}

	return currencies.CostBasis{
		NamespacedID:  models.NamespacedID{ID: c.ID, Namespace: c.Namespace},
		ManagedModel:  models.ManagedModel{CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, DeletedAt: c.DeletedAt},
		CurrencyID:    c.CurrencyID,
		FiatCode:      string(c.FiatCode),
		Rate:          c.Rate,
		EffectiveFrom: c.EffectiveFrom.In(time.UTC),
		EffectiveTo:   effectiveTo,
	}
}

func (a *adapter) ListCustomCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) (pagination.Result[currencies.Currency], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[currencies.Currency], error) {
		q := tx.db.CustomCurrency.Query().
			Where(customcurrency.Namespace(params.Namespace))

		q = filter.ApplyToQuery(q, params.Code, customcurrency.FieldCode)

		order := entutils.GetOrdering(sortx.OrderDefault)
		if !params.Order.IsDefaultValue() {
			order = entutils.GetOrdering(params.Order)
		}
		switch params.OrderBy {
		case currencies.OrderByName:
			q = q.Order(customcurrency.ByName(order...))
		default:
			q = q.Order(customcurrency.ByCode(order...))
		}

		paged, err := q.Paginate(ctx, params.Page)
		if err != nil {
			return pagination.Result[currencies.Currency]{}, fmt.Errorf("failed to list currencies: %w", err)
		}

		return pagination.MapResult(paged, mapCurrencyFromDB), nil
	})
}

func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		curr, err := tx.db.CustomCurrency.Create().
			SetNamespace(params.Namespace).
			SetCode(params.Code).
			SetName(params.Name).
			SetSymbol(params.Symbol).
			Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
			}
			return currencies.Currency{}, fmt.Errorf("failed to create currency: %w", err)
		}

		return mapCurrencyFromDB(curr), nil
	})
}

func (a *adapter) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.CostBasis, error) {
		var effectiveTo *time.Time
		if params.EffectiveTo != nil {
			t := params.EffectiveTo.In(time.UTC)
			effectiveTo = &t
		}

		costBasis, err := tx.db.CurrencyCostBasis.Create().
			SetNamespace(params.Namespace).
			SetCurrencyID(params.CurrencyID).
			SetFiatCode(currencyx.Code(params.FiatCode)).
			SetRate(params.Rate).
			SetEffectiveFrom(params.EffectiveFrom.In(time.UTC)).
			SetNillableEffectiveTo(effectiveTo).
			Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.CostBasis{}, models.NewGenericConflictError(fmt.Errorf("failed to create cost basis: %w", err))
			}
			return currencies.CostBasis{}, fmt.Errorf("failed to create cost basis: %w", err)
		}

		return mapCostBasisFromDB(costBasis), nil
	})
}

func (a *adapter) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) (pagination.Result[currencies.CostBasis], error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (pagination.Result[currencies.CostBasis], error) {
		q := tx.db.CurrencyCostBasis.Query().
			Where(
				currencycostbasis.Namespace(params.Namespace),
				currencycostbasis.CurrencyID(params.CurrencyID),
			).
			Order(currencycostbasis.ByEffectiveFrom(sql.OrderDesc()))

		if params.FilterFiatCode != nil {
			q = q.Where(currencycostbasis.FiatCode(currencyx.Code(*params.FilterFiatCode)))
		}

		paged, err := q.Paginate(ctx, params.Page)
		if err != nil {
			return pagination.Result[currencies.CostBasis]{}, fmt.Errorf("failed to list cost bases: %w", err)
		}

		return pagination.MapResult(paged, mapCostBasisFromDB), nil
	})
}
