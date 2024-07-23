package postgresadapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/internal/credit"
	"github.com/openmeterio/openmeter/internal/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// We implement entuitls.TxUser[T] and entuitls.TxCreator here
// There ought to be a better way....

func (e *grantDBADapter) Tx(ctx context.Context) (context.Context, *entutils.TxDriver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (e *grantDBADapter) WithTx(ctx context.Context, tx *entutils.TxDriver) credit.GrantRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresGrantRepo(txClient.Client())
}

func (e *balanceSnapshotAdapter) Tx(ctx context.Context) (context.Context, *entutils.TxDriver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (e *balanceSnapshotAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) credit.BalanceSnapshotRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresBalanceSnapshotRepo(txClient.Client())
}
