package subscriptionaddonrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// Transaction handling for subscriptionAddonRepo

func (r *subscriptionAddonRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionAddonRepo) Self() *subscriptionAddonRepo {
	return r
}

func (r *subscriptionAddonRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionAddonRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionAddonRepo(txClient.Client())
}

// Transaction handling for subscriptionAddonRateCardRepo

func (r *subscriptionAddonRateCardRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionAddonRateCardRepo) Self() *subscriptionAddonRateCardRepo {
	return r
}

func (r *subscriptionAddonRateCardRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionAddonRateCardRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionAddonRateCardRepo(txClient.Client())
}

// Transaction handling for subscriptionAddonQuantityRepo

func (r *subscriptionAddonQuantityRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *subscriptionAddonQuantityRepo) Self() *subscriptionAddonQuantityRepo {
	return r
}

func (r *subscriptionAddonQuantityRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionAddonQuantityRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionAddonQuantityRepo(txClient.Client())
}
