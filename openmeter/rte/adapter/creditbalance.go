package adapter

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	balancedb "github.com/openmeterio/openmeter/openmeter/ent/db/rtecreditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func balanceScope(key rte.CreditBalanceKey) predicate.RTECreditBalance {
	return balancedb.And(
		balancedb.NamespaceEQ(key.CustomerID.Namespace),
		balancedb.CustomerIDEQ(key.CustomerID.ID),
		balancedb.CurrencyKeyEQ(key.Currency.IdentityKey()),
		balancedb.FeatureScopeEQ(key.FeatureScope()),
		balancedb.DeletedAtIsNil(),
	)
}

func (a *adapter) StoreCreditBalance(ctx context.Context, input rte.StoreCreditBalanceInput) (rte.CreditBalance, error) {
	if err := input.Validate(); err != nil {
		return rte.CreditBalance{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (rte.CreditBalance, error) {
		err := a.db.RTECreditBalance.Create().
			SetNamespace(input.Key.CustomerID.Namespace).
			SetCustomerID(input.Key.CustomerID.ID).
			SetCurrencyKey(input.Key.Currency.IdentityKey()).
			SetFeatureScope(input.Key.FeatureScope()).
			SetCalculatedAt(input.Balance.CalculatedAt).
			SetBooked(input.Balance.Booked).
			SetSpendable(input.Balance.Spendable).
			SetLive(input.Balance.Live).
			SetPending(input.Balance.Pending).
			OnConflict(
				sql.ConflictColumns(balancedb.FieldNamespace, balancedb.FieldCustomerID, balancedb.FieldCurrencyKey, balancedb.FieldFeatureScope),
				sql.ConflictWhere(sql.IsNull(balancedb.FieldDeletedAt)),
				sql.UpdateWhere(sql.LT(balancedb.FieldCalculatedAt, input.Balance.CalculatedAt)),
			).UpdateNewValues().Exec(ctx)
		// PostgreSQL returns no row when the conflict condition retains the
		// existing result. Ent's RETURNING reports that as sql.ErrNoRows.
		if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
			return rte.CreditBalance{}, fmt.Errorf("store credit balance: %w", err)
		}

		// A superseded attempt returns the retained result, not its stale output.
		return a.GetCreditBalance(ctx, input.Key)
	})
}

func (a *adapter) GetCreditBalance(ctx context.Context, key rte.CreditBalanceKey) (rte.CreditBalance, error) {
	if err := key.Validate(); err != nil {
		return rte.CreditBalance{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (rte.CreditBalance, error) {
		entity, err := a.db.RTECreditBalance.Query().Where(balanceScope(key)).Only(ctx)
		if db.IsNotFound(err) {
			return rte.CreditBalance{}, rte.NewCreditBalanceNotCalculatedError()
		}

		if err != nil {
			return rte.CreditBalance{}, fmt.Errorf("get credit balance: %w", err)
		}

		return rte.CreditBalance{
			CalculatedAt: entity.CalculatedAt,
			Booked:       entity.Booked,
			Spendable:    entity.Spendable,
			Live:         entity.Live,
			Pending:      entity.Pending,
		}, nil
	})
}
