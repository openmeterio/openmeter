package appadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type Config struct {
	Client      *entdb.Client
	Marketplace app.MarketplaceAdapter
}

func (c Config) Validate() error {
	if c.Client == nil {
		return errors.New("ent client is required")
	}

	if c.Marketplace == nil {
		return errors.New("marketplace adapter is required")
	}

	return nil
}

func New(config Config) (app.Adapter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	adapter := &adapter{
		db:          config.Client,
		marketplace: config.Marketplace,
	}

	return adapter, nil
}

func NewMarketplaceAdapter() app.MarketplaceAdapter {
	return DefaultMarketplace()
}

var _ app.Adapter = (*adapter)(nil)

type adapter struct {
	db          *entdb.Client
	marketplace app.MarketplaceAdapter
}

// Tx implements entutils.TxCreator interface
func (e adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := e.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}
