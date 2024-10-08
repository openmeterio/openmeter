package appadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
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

type MarketplaceConfig struct {
	BaseURL string
}

func (c MarketplaceConfig) Validate() error {
	if c.BaseURL == "" {
		return errors.New("base url is required")
	}

	return nil
}

type Marketplace struct {
	registry map[appentitybase.AppType]appentity.RegistryItem
	baseURL  string
}

func NewMarketplaceAdapter(config MarketplaceConfig) app.MarketplaceAdapter {
	return Marketplace{
		registry: map[appentitybase.AppType]appentity.RegistryItem{},
		baseURL:  config.BaseURL,
	}
}

var _ app.Adapter = (*adapter)(nil)

type adapter struct {
	db          *entdb.Client
	marketplace app.MarketplaceAdapter
	baseURL     string
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
