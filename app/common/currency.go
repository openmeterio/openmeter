package common

import (
	"fmt"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

var Currency = wire.NewSet(
	NewCurrencyAdapter,
	NewCurrencyService,
)

func NewCurrencyAdapter(db *entdb.Client) (currencies.Repository, error) {
	repo, err := currencyadapter.New(currencyadapter.Config{
		Client: db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create currency adapter: %w", err)
	}

	return repo, nil
}

func NewCurrencyService(repo currencies.Repository) (currencies.Service, error) {
	s, err := service.New(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create currency service: %w", err)
	}

	return s, nil
}
