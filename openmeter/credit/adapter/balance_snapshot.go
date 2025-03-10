package adapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	db_balancesnapshot "github.com/openmeterio/openmeter/openmeter/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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
	return entutils.TransactingRepoWithNoValue(ctx, b, func(ctx context.Context, rep *balanceSnapshotRepo) error {
		return rep.db.BalanceSnapshot.Update().
			Where(db_balancesnapshot.OwnerID(owner.ID), db_balancesnapshot.Namespace(owner.Namespace), db_balancesnapshot.AtGT(at)).
			SetDeletedAt(clock.Now()).
			Exec(ctx)
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
		commands := make([]*db.BalanceSnapshotCreate, 0, len(balances))
		for _, balance := range balances {
			command := rep.db.BalanceSnapshot.Create().
				SetNamespace(owner.Namespace).
				SetOwnerID(owner.ID).
				SetBalance(balance.Balance()).
				SetAt(balance.At).
				SetGrantBalances(balance.Balances).
				SetOverage(balance.Overage).
				SetUsage(&balance.Usage)
			commands = append(commands, command)
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
	if entity.Usage != nil {
		s.Usage = *entity.Usage
	}
	return s
}
