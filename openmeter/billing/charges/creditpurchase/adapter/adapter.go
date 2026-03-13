package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	MetaAdapter meta.Adapter
	Client      *entdb.Client
	Logger      *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.Logger == nil {
		return errors.New("logger is required")
	}

	if c.MetaAdapter == nil {
		return errors.New("meta adapter is required")
	}

	return nil
}

func New(config Config) (creditpurchase.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:          config.Client,
		logger:      config.Logger,
		metaAdapter: config.MetaAdapter,
	}, nil
}

var _ creditpurchase.Adapter = (*adapter)(nil)

type adapter struct {
	db          *entdb.Client
	logger      *slog.Logger
	metaAdapter meta.Adapter
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
		db:          txDb.Client(),
		logger:      a.logger,
		metaAdapter: a.metaAdapter,
	}
}

func (a *adapter) Self() *adapter {
	return a
}
