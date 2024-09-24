package adapter

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// We implement entuitls.TxUser[T] and entuitls.TxCreator here
// There ought to be a better way....

func (e *entitlementDBAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (e *entitlementDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *entitlementDBAdapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresEntitlementRepo(txClient.Client())
}

func (u *usageResetDBAdapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := u.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (u *usageResetDBAdapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *usageResetDBAdapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresUsageResetRepo(txClient.Client())
}
