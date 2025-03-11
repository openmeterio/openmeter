package appadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client  *entdb.Client
	BaseURL string
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.BaseURL == "" {
		return errors.New("base url is required")
	}

	return nil
}

func New(config Config) (app.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	adapter := &adapter{
		db:       config.Client,
		registry: map[app.AppType]app.RegistryItem{},
		baseURL:  config.BaseURL,
	}

	return adapter, nil
}

var _ app.Adapter = (*adapter)(nil)

type adapter struct {
	db       *entdb.Client
	registry map[app.AppType]app.RegistryItem
	baseURL  string
}

// Tx implements entutils.TxCreator interface
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
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return &adapter{
		db:       txClient.Client(),
		registry: a.registry,
		baseURL:  a.baseURL,
	}
}

func (a *adapter) Self() *adapter {
	return a
}
