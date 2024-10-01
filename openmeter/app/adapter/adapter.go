package appadapter

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
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
		db:          entcontext.NewClient(config.Client),
		marketplace: config.Marketplace,
	}

	return adapter, nil
}

func NewMarketplaceAdapter() app.MarketplaceAdapter {
	return DefaultMarketplace()
}

var _ app.Adapter = (*adapter)(nil)

type adapter struct {
	db          entcontext.DB
	marketplace app.MarketplaceAdapter
}

func (a *adapter) DB() entcontext.DB {
	return a.db
}
