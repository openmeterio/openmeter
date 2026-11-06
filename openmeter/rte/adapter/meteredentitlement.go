package adapter

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	balancedb "github.com/openmeterio/openmeter/openmeter/ent/db/rtemeteredentitlementbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type meteredEntitlementStore struct {
	*adapter
}

func NewMeteredEntitlementBalanceStore(config Config) (rte.Store[meteredentitlement.Key, meteredentitlement.Balance], error) {
	adapter, err := newAdapter(config)
	if err != nil {
		return nil, err
	}

	return &meteredEntitlementStore{adapter: adapter}, nil
}

func meteredBalanceScope(key meteredentitlement.Key) predicate.RTEMeteredEntitlementBalance {
	return balancedb.And(
		balancedb.NamespaceEQ(key.EntitlementID.Namespace),
		balancedb.EntitlementIDEQ(key.EntitlementID.ID),
		balancedb.DeletedAtIsNil(),
	)
}

func (a *meteredEntitlementStore) Store(ctx context.Context, input rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]) (rte.State[meteredentitlement.Balance], error) {
	if err := input.Validate(); err != nil {
		return rte.State[meteredentitlement.Balance]{}, err
	}

	encoded, err := json.Marshal(input.State.Value)
	if err != nil {
		return rte.State[meteredentitlement.Balance]{}, fmt.Errorf("encode metered entitlement balance: %w", err)
	}

	return entutils.TransactingRepo(ctx, a.adapter, func(ctx context.Context, a *adapter) (rte.State[meteredentitlement.Balance], error) {
		err := a.db.RTEMeteredEntitlementBalance.Create().
			SetNamespace(input.Key.EntitlementID.Namespace).
			SetEntitlementID(input.Key.EntitlementID.ID).
			SetCalculatedAt(input.State.CalculatedAt).
			SetBalance(encoded).
			OnConflict(
				sql.ConflictColumns(balancedb.FieldNamespace, balancedb.FieldEntitlementID),
				sql.ConflictWhere(sql.IsNull(balancedb.FieldDeletedAt)),
				sql.UpdateWhere(sql.LT(balancedb.FieldCalculatedAt, input.State.CalculatedAt)),
			).UpdateNewValues().Exec(ctx)
		// PostgreSQL returns no row when the conflict condition retains the
		// existing result. Ent's RETURNING reports that as sql.ErrNoRows.
		if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
			return rte.State[meteredentitlement.Balance]{}, fmt.Errorf("store metered entitlement balance: %w", err)
		}

		// A superseded attempt returns the retained result, not its stale output.
		return (&meteredEntitlementStore{adapter: a}).Get(ctx, input.Key)
	})
}

func (a *meteredEntitlementStore) Get(ctx context.Context, key meteredentitlement.Key) (rte.State[meteredentitlement.Balance], error) {
	if err := key.Validate(); err != nil {
		return rte.State[meteredentitlement.Balance]{}, err
	}

	return entutils.TransactingRepo(ctx, a.adapter, func(ctx context.Context, a *adapter) (rte.State[meteredentitlement.Balance], error) {
		entity, err := a.db.RTEMeteredEntitlementBalance.Query().Where(meteredBalanceScope(key)).Only(ctx)
		if db.IsNotFound(err) {
			return rte.State[meteredentitlement.Balance]{}, rte.NewNotCalculatedError()
		}

		if err != nil {
			return rte.State[meteredentitlement.Balance]{}, fmt.Errorf("get metered entitlement balance: %w", err)
		}

		result := rte.State[meteredentitlement.Balance]{CalculatedAt: entity.CalculatedAt}
		if err := json.Unmarshal(entity.Balance, &result.Value); err != nil {
			return rte.State[meteredentitlement.Balance]{}, fmt.Errorf("decode metered entitlement balance: %w", err)
		}

		return result, nil
	})
}
