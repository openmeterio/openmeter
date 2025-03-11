package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (r *subscriptionRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionRepo) Self() *subscriptionRepo {
	return r
}

func (r *subscriptionRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionRepo(txClient.Client())
}

func (r *subscriptionPhaseRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionPhaseRepo) Self() *subscriptionPhaseRepo {
	return r
}

func (r *subscriptionPhaseRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionPhaseRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionPhaseRepo(txClient.Client())
}

func (r *subscriptionItemRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionItemRepo) Self() *subscriptionItemRepo {
	return r
}

func (r *subscriptionItemRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionItemRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionItemRepo(txClient.Client())
}
