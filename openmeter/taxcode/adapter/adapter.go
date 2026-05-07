package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Config)(nil)

type Config struct {
	Client *entdb.Client
	Logger *slog.Logger
}

func (c Config) Validate() error {
	var errs []error

	if c.Client == nil {
		errs = append(errs, errors.New("postgres client is required"))
	}

	if c.Logger == nil {
		errs = append(errs, errors.New("logger is required"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func New(config Config) (taxcode.Repository, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

var _ taxcode.Repository = (*adapter)(nil)

type adapter struct {
	db *entdb.Client

	logger *slog.Logger
}

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	ctx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}

	return ctx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

	return &adapter{
		db:     txClient.Client(),
		logger: a.logger,
	}
}

func (a *adapter) Self() *adapter {
	return a
}
