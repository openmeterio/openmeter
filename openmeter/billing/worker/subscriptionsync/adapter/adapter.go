package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *entdb.Client
}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, errors.New("client is required"))
	}

	return errors.Join(errs...)
}

type adapter struct {
	db *entdb.Client
}

func New(config Config) (subscriptionsync.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db: config.Client,
	}, nil
}

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &adapter{
		db: txDb.Client(),
	}
}

func (a *adapter) Self() *adapter {
	return a
}
