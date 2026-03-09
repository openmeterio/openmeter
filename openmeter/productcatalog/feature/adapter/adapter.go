package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

// New creates a new feature adapter.
func New(config Config) (feature.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:     config.Client,
		logger: config.Logger,
	}, nil
}

// NewPostgresFeatureRepo is an alias for New that matches the old constructor signature, kept for backward compatibility.
func NewPostgresFeatureRepo(db *entdb.Client, logger *slog.Logger) feature.Adapter {
	a, err := New(Config{
		Client: db,
		Logger: logger,
	})
	if err != nil {
		// This should never happen since the old constructor didn't return an error,
		// but panic is acceptable for backward compat wrappers with known-good inputs.
		panic(fmt.Sprintf("failed to create feature adapter: %v", err))
	}
	return a
}

var _ feature.Adapter = (*adapter)(nil)

type adapter struct {
	db     *entdb.Client
	logger *slog.Logger
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

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) feature.Adapter {
	txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewPostgresFeatureRepo(txClient.Client(), a.logger)
}

func (a *adapter) Self() feature.Adapter {
	return a
}
