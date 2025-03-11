package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// We implement entuitls.TxUser[T] and entuitls.TxCreator here
// There ought to be a better way....

func (e *grantDBADapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (e *grantDBADapter) WithTx(ctx context.Context, tx *entutils.TxDriver) grant.Repo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresGrantRepo(txClient.Client())
}

func (e *grantDBADapter) Self() grant.Repo {
	return e
}

func (e *balanceSnapshotRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (e *balanceSnapshotRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *balanceSnapshotRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresBalanceSnapshotRepo(txClient.Client())
}

func (e *balanceSnapshotRepo) Self() *balanceSnapshotRepo {
	return e
}
