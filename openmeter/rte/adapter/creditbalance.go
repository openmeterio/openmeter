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
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type creditBalanceStore struct {
	*adapter
}

func NewCreditBalanceStore(config Config) (rte.Store[creditbalance.Key, creditbalance.Balance], error) {
	adapter, err := newAdapter(config)
	if err != nil {
		return nil, err
	}

	return &creditBalanceStore{adapter: adapter}, nil
}

func balanceScope(key creditbalance.Key) predicate.RTECreditBalance {
	return balancedb.And(
		balancedb.NamespaceEQ(key.CustomerID.Namespace),
		balancedb.CustomerIDEQ(key.CustomerID.ID),
		balancedb.CurrencyKeyEQ(key.Currency.IdentityKey()),
		balancedb.FeatureScopeEQ(key.FeatureScope()),
		balancedb.DeletedAtIsNil(),
	)
}

func (a *creditBalanceStore) Store(ctx context.Context, input rte.StoreInput[creditbalance.Key, creditbalance.Balance]) (rte.StoreResult[creditbalance.Balance], error) {
	if err := input.Validate(); err != nil {
		return rte.StoreResult[creditbalance.Balance]{}, err
	}

	return entutils.TransactingRepo(ctx, a.adapter, func(ctx context.Context, a *adapter) (rte.StoreResult[creditbalance.Balance], error) {
		err := a.db.RTECreditBalance.Create().
			SetNamespace(input.Key.CustomerID.Namespace).
			SetCustomerID(input.Key.CustomerID.ID).
			SetCurrencyKey(input.Key.Currency.IdentityKey()).
			SetFeatureScope(input.Key.FeatureScope()).
			SetCalculatedAt(input.State.CalculatedAt).
			SetBooked(input.State.Value.Booked).
			SetSpendable(input.State.Value.Spendable).
			SetLive(input.State.Value.Live).
			SetPending(input.State.Value.Pending).
			OnConflict(
				sql.ConflictColumns(balancedb.FieldNamespace, balancedb.FieldCustomerID, balancedb.FieldCurrencyKey, balancedb.FieldFeatureScope),
				sql.ConflictWhere(sql.IsNull(balancedb.FieldDeletedAt)),
				sql.UpdateWhere(sql.LT(balancedb.FieldCalculatedAt, input.State.CalculatedAt)),
			).UpdateNewValues().Exec(ctx)
		// PostgreSQL returns no row when the conflict condition retains the
		// existing result. Ent's RETURNING reports that as sql.ErrNoRows.
		if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
			return rte.StoreResult[creditbalance.Balance]{}, fmt.Errorf("store credit balance: %w", err)
		}

		// A superseded attempt returns the retained result, not its stale output.
		state, readErr := (&creditBalanceStore{adapter: a}).Get(ctx, input.Key)
		if readErr != nil {
			return rte.StoreResult[creditbalance.Balance]{}, readErr
		}

		return rte.StoreResult[creditbalance.Balance]{State: state, Updated: err == nil}, nil
	})
}

func (a *creditBalanceStore) Get(ctx context.Context, key creditbalance.Key) (rte.State[creditbalance.Balance], error) {
	if err := key.Validate(); err != nil {
		return rte.State[creditbalance.Balance]{}, err
	}

	return entutils.TransactingRepo(ctx, a.adapter, func(ctx context.Context, a *adapter) (rte.State[creditbalance.Balance], error) {
		entity, err := a.db.RTECreditBalance.Query().Where(balanceScope(key)).Only(ctx)
		if db.IsNotFound(err) {
			return rte.State[creditbalance.Balance]{}, rte.NewNotCalculatedError()
		}

		if err != nil {
			return rte.State[creditbalance.Balance]{}, fmt.Errorf("get credit balance: %w", err)
		}

		return rte.State[creditbalance.Balance]{
			CalculatedAt: entity.CalculatedAt,
			Value: creditbalance.Balance{
				Booked:    entity.Booked,
				Spendable: entity.Spendable,
				Live:      entity.Live,
				Pending:   entity.Pending,
			},
		}, nil
	})
}
