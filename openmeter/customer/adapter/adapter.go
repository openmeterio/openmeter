package adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client *entdb.Client
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

func New(config Config) (customer.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &adapter{
		db:        config.Client,
		logger:    config.Logger,
		observers: &[]appobserver.Observer[customerentity.Customer]{},
	}, nil
}

var (
	_ customer.Adapter                               = (*adapter)(nil)
	_ appobserver.Publisher[customerentity.Customer] = (*adapter)(nil)
)

type adapter struct {
	db *entdb.Client
	// It is a reference so we can pass it down in WithTx
	observers *[]appobserver.Observer[customerentity.Customer]

	logger *slog.Logger
}

// Tx implements entutils.TxCreator interface
func (a adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &adapter{
		db:        txClient.Client(),
		logger:    a.logger,
		observers: a.observers,
	}
}
