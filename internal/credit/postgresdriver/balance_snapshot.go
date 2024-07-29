package postgresadapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	balancesnapshot "github.com/openmeterio/openmeter/internal/credit/balance_snapshot"
	"github.com/openmeterio/openmeter/internal/credit/grant"
	"github.com/openmeterio/openmeter/internal/ent/db"
	db_balancesnapshot "github.com/openmeterio/openmeter/internal/ent/db/balancesnapshot"
	"github.com/openmeterio/openmeter/pkg/clock"
)

// naive implementation of the BalanceSnapshotConnector
type balanceSnapshotAdapter struct {
	db *db.Client
}

func NewPostgresBalanceSnapshotRepo(db *db.Client) balancesnapshot.BalanceSnapshotRepo {
	return &balanceSnapshotAdapter{
		db: db,
	}
}

func (b *balanceSnapshotAdapter) InvalidateAfter(ctx context.Context, owner grant.NamespacedGrantOwner, at time.Time) error {
	return b.db.BalanceSnapshot.Update().
		Where(db_balancesnapshot.OwnerID(string(owner.ID)), db_balancesnapshot.Namespace(owner.Namespace), db_balancesnapshot.AtGT(at)).
		SetDeletedAt(clock.Now()).
		Exec(ctx)
}

func (b *balanceSnapshotAdapter) GetLatestValidAt(ctx context.Context, owner grant.NamespacedGrantOwner, at time.Time) (balancesnapshot.GrantBalanceSnapshot, error) {
	res, err := b.db.BalanceSnapshot.Query().
		Where(
			db_balancesnapshot.OwnerID(string(owner.ID)),
			db_balancesnapshot.Namespace(owner.Namespace),
			db_balancesnapshot.AtLTE(at),
			db_balancesnapshot.DeletedAtIsNil(),
		).
		// in case there were multiple snapshots for the same time return the newest one
		Order(db_balancesnapshot.ByAt(sql.OrderDesc()), db_balancesnapshot.ByUpdatedAt(sql.OrderDesc())).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return balancesnapshot.GrantBalanceSnapshot{}, &balancesnapshot.GrantBalanceNoSavedBalanceForOwnerError{Owner: owner, Time: at}
		}
		return balancesnapshot.GrantBalanceSnapshot{}, err
	}

	return mapBalanceSnapshotEntity(res), nil
}

func (b *balanceSnapshotAdapter) Save(ctx context.Context, owner grant.NamespacedGrantOwner, balances []balancesnapshot.GrantBalanceSnapshot) error {
	commands := make([]*db.BalanceSnapshotCreate, 0, len(balances))
	for _, balance := range balances {
		command := b.db.BalanceSnapshot.Create().
			SetOwnerID(string(owner.ID)).
			SetNamespace(owner.Namespace).
			SetBalance(balance.Balance()).
			SetAt(balance.At).
			SetGrantBalances(balance.Balances).
			SetOverage(balance.Overage)
		commands = append(commands, command)
	}
	_, err := b.db.BalanceSnapshot.CreateBulk(commands...).Save(ctx)
	return err
}

func mapBalanceSnapshotEntity(entity *db.BalanceSnapshot) balancesnapshot.GrantBalanceSnapshot {
	return balancesnapshot.GrantBalanceSnapshot{
		Balances: entity.GrantBalances,
		Overage:  entity.Overage,
		At:       entity.At.In(time.UTC),
	}
}
