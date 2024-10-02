package appservice

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/app"
)

var _ app.Service = (*Service)(nil)

type Service struct {
	adapter     app.Adapter
	marketplace app.MarketplaceAdapter
}

type Config struct {
	Adapter     app.Adapter
	Marketplace app.MarketplaceAdapter
}

func (c Config) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter cannot be null")
	}

	if c.Marketplace == nil {
		return errors.New("marketplace cannot be null")
	}

	return nil
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		adapter:     config.Adapter,
		marketplace: config.Marketplace,
	}, nil
}
