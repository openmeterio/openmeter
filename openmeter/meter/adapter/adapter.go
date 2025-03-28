package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *db.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("postgres client is required")
	}

	if c.Logger == nil {
		return errors.New("logger must not be nil")
	}

	return nil
}

func New(config Config) (*Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ meter.Service = (*Adapter)(nil)

type Adapter struct {
	db     *db.Client
	logger *slog.Logger
}

// Tx implements entutils.TxCreator interface
func (a *Adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *Adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *Adapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &Adapter{
		db:     txClient.Client(),
		logger: a.logger,
	}
}

func (a *Adapter) Self() *Adapter {
	return a
}
