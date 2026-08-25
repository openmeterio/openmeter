package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_balancesnapshot "github.com/openmeterio/openmeter/openmeter/ent/db/balancesnapshot"
	db_entitlement "github.com/openmeterio/openmeter/openmeter/ent/db/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// naive implementation of the BalanceSnapshotConnector
type balanceSnapshotRepo struct {
	db *db.Client
}

func NewPostgresBalanceSnapshotRepo(db *db.Client) *balanceSnapshotRepo {
	return &balanceSnapshotRepo{
		db: db,
	}
}

func (b *balanceSnapshotRepo) InvalidateAfter(ctx context.Context, owner models.NamespacedID, at time.Time) error {
	return transaction.RunWithNoValue(ctx, b, func(ctx context.Context) error {
		return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error {
			updated, err := rep.db.Entitlement.Update().
				Where(
					db_entitlement.ID(owner.ID),
					db_entitlement.Namespace(owner.Namespace),
				).
				AddBalanceSnapshotInvalidationVersion(1).
				Save(ctx)
			if err != nil {
				return err
			}
			if updated == 0 {
				return grant.NewOwnerNotFoundError(owner, "entitlement")
			}

			return rep.db.BalanceSnapshot.Update().
				Where(
					db_balancesnapshot.OwnerID(owner.ID),
					db_balancesnapshot.Namespace(owner.Namespace),
					db_balancesnapshot.AtGT(at),
					db_balancesnapshot.DeletedAtIsNil(),
				).
				SetDeletedAt(clock.Now()).
				Exec(ctx)
		})
	})
}

func (b *balanceSnapshotRepo) GetInvalidationVersion(ctx context.Context, owner models.NamespacedID) (balance.SnapshotInvalidationVersion, error) {
	return entutils.TransactingRepo(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) (balance.SnapshotInvalidationVersion, error) {
		entitlement, err := rep.db.Entitlement.Query().
			Where(
				db_entitlement.ID(owner.ID),
				db_entitlement.Namespace(owner.Namespace),
			).
			Select(db_entitlement.FieldBalanceSnapshotInvalidationVersion).
			Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return 0, grant.NewOwnerNotFoundError(owner, "entitlement")
			}

			return 0, err
		}

		return balance.SnapshotInvalidationVersion(entitlement.BalanceSnapshotInvalidationVersion), nil
	})
}

func (b *balanceSnapshotRepo) GetLatestValidAt(ctx context.Context, owner models.NamespacedID, at time.Time) (balance.Snapshot, error) {
	return entutils.TransactingRepo(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) (balance.Snapshot, error) {
		res, err := rep.db.BalanceSnapshot.Query().
			Where(
				db_balancesnapshot.OwnerID(owner.ID),
				db_balancesnapshot.Namespace(owner.Namespace),
				db_balancesnapshot.AtLTE(at),
				db_balancesnapshot.DeletedAtIsNil(),
				db_balancesnapshot.UsageSnapshotNotNil(),
			).
			// in case there were multiple snapshots for the same time return the newest one
			Order(db_balancesnapshot.ByAt(sql.OrderDesc()), db_balancesnapshot.ByUpdatedAt(sql.OrderDesc())).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return balance.Snapshot{}, &balance.NoSavedBalanceForOwnerError{Owner: owner, Time: at}
			}
			return balance.Snapshot{}, err
		}

		return mapBalanceSnapshotEntity(res), nil
	})
}

func (b *balanceSnapshotRepo) Save(ctx context.Context, owner models.NamespacedID, balances []balance.Snapshot) error {
	return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error {
		if len(balances) == 0 {
			return nil
		}

		commands := make([]*db.BalanceSnapshotCreate, 0, len(balances))
		for _, snapshot := range balances {
			if snapshot.UsageSnapshot == nil {
				return fmt.Errorf("cannot save incomplete balance snapshot at %s", snapshot.At)
			}

			// Keep writing the legacy usage representation for compatibility
			// with old readers during the rolling migration.
			command := rep.db.BalanceSnapshot.Create().
				SetNamespace(owner.Namespace).
				SetOwnerID(owner.ID).
				SetBalance(snapshot.Balance()).
				SetAt(snapshot.At).
				SetGrantBalances(snapshot.Balances).
				SetOverage(snapshot.Overage).
				SetUsage(&snapshot.Usage).
				SetUsageSnapshot(snapshot.UsageSnapshot)
			// Record the conversion regime this snapshot was computed under (OM-400) so
			// the resume path can refuse to reuse it under a different regime. Pointer
			// GoType has no SetNillable*, so nil-guard the set (nil = raw).
			if snapshot.UnitConfig != nil {
				command = command.SetUnitConfig(snapshot.UnitConfig)
			}
			commands = append(commands, command)
		}

		if err := ensureEntitlementOwnerExists(ctx, rep.db, owner); err != nil {
			return err
		}

		_, err := rep.db.BalanceSnapshot.CreateBulk(commands...).Save(ctx)
		return err
	})
}

func mapBalanceSnapshotEntity(entity *db.BalanceSnapshot) balance.Snapshot {
	s := balance.Snapshot{
		Balances: entity.GrantBalances,
		Overage:  entity.Overage,
		At:       entity.At.In(time.UTC),
	}
	if entity.UsageSnapshot != nil {
		s.UsageSnapshot = entity.UsageSnapshot
	}
	if entity.Usage != nil {
		// Hydrate legacy usage only so subsequent snapshots remain readable by
		// old binaries during the rolling migration.
		s.Usage = *entity.Usage
	}
	if entity.UnitConfig != nil {
		s.UnitConfig = entity.UnitConfig
	}
	return s
}
