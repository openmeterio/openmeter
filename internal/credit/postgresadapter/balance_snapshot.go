package postgresadapter

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db"
	db_balancesnapshot "github.com/openmeterio/openmeter/internal/credit/postgresadapter/ent/db/balancesnapshot"
)

type BalanceSnapshotConfig struct {
	// Enabled is a flag to enable or disable the balance snapshot feature, if disabled
	// we will still write the data to the database, but we will never utilize the snaphots created
	Enabled bool
}

// naive implementation of the BalanceSnapshotConnector
type balanceSnapshotAdapter struct {
	db     *db.Client
	config BalanceSnapshotConfig
}

func NewPostgresBalanceSnapshotRepo(db *db.Client, config BalanceSnapshotConfig) credit.BalanceSnapshotConnector {
	return &balanceSnapshotAdapter{
		db:     db,
		config: config,
	}
}

func (b *balanceSnapshotAdapter) InvalidateAfter(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) error {
	return b.db.BalanceSnapshot.Update().
		Where(db_balancesnapshot.OwnerID(owner.ID), db_balancesnapshot.Namespace(owner.Namespace), db_balancesnapshot.AtGT(at)).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (b *balanceSnapshotAdapter) GetLatestValidAt(ctx context.Context, owner credit.NamespacedGrantOwner, at time.Time) (credit.GrantBalanceSnapshot, error) {
	if !b.config.Enabled {
		return credit.GrantBalanceSnapshot{}, &credit.GrantBalanceNoSavedBalanceForOwnerError{Owner: owner, Time: at}
	}

	res, err := b.db.BalanceSnapshot.Query().
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
			return credit.GrantBalanceSnapshot{}, &credit.GrantBalanceNoSavedBalanceForOwnerError{Owner: owner, Time: at}
		}
		return credit.GrantBalanceSnapshot{}, err
	}

	return mapBalanceSnapshotEntity(res), nil
}

func (b *balanceSnapshotAdapter) Save(ctx context.Context, owner credit.NamespacedGrantOwner, balances []credit.GrantBalanceSnapshot) error {
	commands := make([]*db.BalanceSnapshotCreate, 0, len(balances))
	for _, balance := range balances {
		command := b.db.BalanceSnapshot.Create().
			SetOwnerID(owner.ID).
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

func mapBalanceSnapshotEntity(entity *db.BalanceSnapshot) credit.GrantBalanceSnapshot {
	return credit.GrantBalanceSnapshot{
		Balances: entity.GrantBalances,
		Overage:  entity.Overage,
		At:       entity.At.In(time.UTC),
	}
}
