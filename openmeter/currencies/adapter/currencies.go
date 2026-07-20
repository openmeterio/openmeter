package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/currencycostbasis"
	"github.com/openmeterio/openmeter/openmeter/ent/db/customcurrency"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func mapCurrencyFromDB(c *entdb.CustomCurrency) (currencies.Currency, error) {
	curr, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(c.Code).
		WithName(c.Name).
		WithSymbol(c.Symbol).
		WithPrecision(c.Precision).
		WithDecimalMark(c.DecimalMark).
		WithThousandsSeparator(c.ThousandsSeparator).
		Build()
	if err != nil {
		return currencies.Currency{}, fmt.Errorf("failed to map currency from database: %w", err)
	}

	var costBasisList []currencies.CostBasis

	for _, cb := range c.Edges.CostBasisHistory {
		if cb != nil {
			costBasisList = append(costBasisList, mapCostBasisFromDB(cb))
		}
	}

	return currencies.Currency{
		ManagedModel: models.ManagedModel{
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			DeletedAt: c.DeletedAt,
		},
		NamespacedID: models.NamespacedID{
			ID:        c.ID,
			Namespace: c.Namespace,
		},
		Currency: curr,
		CostBasis: func() *[]currencies.CostBasis {
			if len(costBasisList) > 0 {
				return &costBasisList
			}

			return nil
		}(),
	}, nil
}

func mapCostBasisFromDB(c *entdb.CurrencyCostBasis) currencies.CostBasis {
	var effectiveTo *time.Time

	if c.EffectiveTo != nil {
		t := c.EffectiveTo.In(time.UTC)
		effectiveTo = &t
	}

	return currencies.CostBasis{
		ManagedModel: models.ManagedModel{
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			DeletedAt: c.DeletedAt,
		},
		NamespacedID: models.NamespacedID{
			ID:        c.ID,
			Namespace: c.Namespace,
		},
		CostBasis: currencyx.CostBasis{
			FiatCode:      c.FiatCode,
			Rate:          c.Rate,
			EffectiveFrom: c.EffectiveFrom.In(time.UTC),
			EffectiveTo:   effectiveTo,
		},
		CurrencyID: c.CurrencyID,
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

		return pagination.MapResultErr(paged, mapCurrencyFromDB)
	})
}

func (a *adapter) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		q := tx.db.CustomCurrency.Create().
			SetNamespace(params.Namespace).
			SetCode(params.Code).
			SetName(params.Name).
			SetPrecision(params.Precision).
			SetNillableSymbol(lo.EmptyableToPtr(params.Symbol)).
			SetDecimalMark(params.DecimalMark).
			SetThousandsSeparator(params.ThousandsSeparator)

		curr, err := q.Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code))
			}

			return currencies.Currency{}, fmt.Errorf("failed to create currency: %w", err)
		}

		return mapCurrencyFromDB(curr)
	})
}

func (a *adapter) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (currencies.CostBasis, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.CostBasis, error) {
		if params.EffectiveFrom == nil {
			return currencies.CostBasis{}, models.NewGenericValidationError(fmt.Errorf("effective_from must be set"))
		}

		updateQuery := tx.db.CurrencyCostBasis.Update().
			Where(
				currencycostbasis.Namespace(params.Namespace),
				currencycostbasis.CurrencyID(params.CurrencyID),
				currencycostbasis.FiatCode(params.FiatCode),
				currencycostbasis.DeletedAtIsNil(),
				currencycostbasis.EffectiveFromLTE(params.EffectiveFrom.In(time.UTC)),
				currencycostbasis.Or(
					currencycostbasis.EffectiveToIsNil(),
					currencycostbasis.EffectiveToGT(params.EffectiveFrom.In(time.UTC)),
				),
			).
			SetNillableEffectiveTo(lo.ToPtr(params.EffectiveFrom.In(time.UTC)))

		if err := updateQuery.Exec(ctx); err != nil {
			return currencies.CostBasis{}, fmt.Errorf("failed to archive cost basis: %w", err)
		}

		var effectiveTo *time.Time

		if params.EffectiveTo != nil {
			t := params.EffectiveTo.In(time.UTC)
			effectiveTo = &t
		}

		createQuery := tx.db.CurrencyCostBasis.Create().
			SetNamespace(params.Namespace).
			SetCurrencyID(params.CurrencyID).
			SetFiatCode(params.FiatCode).
			SetRate(params.Rate).
			SetEffectiveFrom(params.EffectiveFrom.In(time.UTC)).
			SetNillableEffectiveTo(effectiveTo)

		costBasis, err := createQuery.Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return currencies.CostBasis{}, models.NewGenericConflictError(fmt.Errorf("failed to create cost basis: %w", err))
			}

			return currencies.CostBasis{}, fmt.Errorf("failed to create cost basis: %w", err)
		}

		return mapCostBasisFromDB(costBasis), nil
	})
}

func (a *adapter) GetCostBasis(ctx context.Context, params currencies.GetCostBasisInput) (currencies.CostBasis, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.CostBasis, error) {
		q := tx.db.CurrencyCostBasis.Query().
			Where(
				currencycostbasis.Namespace(params.Namespace),
				currencycostbasis.ID(params.ID),
			)

		if params.CustomCurrency {
			q = q.WithCurrency()
		}

		costBasis, err := q.Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return currencies.CostBasis{}, models.NewGenericNotFoundError(fmt.Errorf("cost basis %q not found", params.ID))
			}

			return currencies.CostBasis{}, fmt.Errorf("failed to get cost basis: %w", err)
		}

		result := mapCostBasisFromDB(costBasis)

		if params.CustomCurrency {
			customCurrency, err := mapCurrencyFromDB(costBasis.Edges.Currency)
			if err != nil {
				return currencies.CostBasis{}, fmt.Errorf("failed to map custom currency: %w", err)
			}

			result.CustomCurrency = &customCurrency
		}

		return result, nil
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
			q = q.Where(currencycostbasis.FiatCode(*params.FilterFiatCode))
		}

		paged, err := q.Paginate(ctx, params.Page)
		if err != nil {
			return pagination.Result[currencies.CostBasis]{}, fmt.Errorf("failed to list cost bases: %w", err)
		}

		return pagination.MapResult(paged, mapCostBasisFromDB), nil
	})
}

func (a *adapter) GetCurrency(ctx context.Context, params currencies.GetCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		at := clock.Now()

		qetQuery := tx.db.CustomCurrency.Query().
			Where(
				customcurrency.Namespace(params.Namespace),
				customcurrency.ID(params.ID),
				customcurrency.Or(
					customcurrency.DeletedAtIsNil(),
					customcurrency.DeletedAtGTE(at),
				),
			)

		c, err := qetQuery.First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return currencies.Currency{}, models.NewGenericNotFoundError(
					fmt.Errorf("currency with id %s not found", params.ID),
				)
			}

			return currencies.Currency{}, fmt.Errorf("failed to get currency: %w", err)
		}

		curr, err := mapCurrencyFromDB(c)
		if err != nil {
			return currencies.Currency{}, fmt.Errorf("failed to map currency from database: %w", err)
		}

		if params.CostBasis {
			if c.DeletedAt != nil {
				at = *c.DeletedAt
			}

			costBasisQuery := tx.db.CurrencyCostBasis.Query().
				Where(
					currencycostbasis.Namespace(params.Namespace),
					currencycostbasis.CurrencyID(params.ID),
					currencycostbasis.EffectiveFromLTE(at),
					currencycostbasis.Or(
						currencycostbasis.EffectiveToIsNil(),
						currencycostbasis.EffectiveToGT(at),
					),
				)

			cbs, err := costBasisQuery.All(ctx)
			if err != nil {
				if entdb.IsNotFound(err) {
					return currencies.Currency{}, models.NewGenericNotFoundError(
						fmt.Errorf("currency with id %s not found", params.ID),
					)
				}

				return currencies.Currency{}, fmt.Errorf("failed to get currency: %w", err)
			}

			curr.CostBasis = lo.ToPtr(
				lo.Map[*entdb.CurrencyCostBasis, currencies.CostBasis](cbs,
					func(item *entdb.CurrencyCostBasis, _ int) currencies.CostBasis {
						return mapCostBasisFromDB(item)
					},
				),
			)
		}

		return curr, nil
	})
}
