package common

import (
	"fmt"
	"log/slog"

	"github.com/google/wire"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currencyAdapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	"github.com/openmeterio/openmeter/openmeter/currencies/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
)

var Currency = wire.NewSet(
	NewCurrencyService,
)

func NewCurrencyService(logger *slog.Logger, db *entdb.Client) (currencies.CurrencyService, error) {
	adapter, err := currencyAdapter.New(currencyAdapter.Config{
		Client: db,
		Logger: logger.WithGroup("currency.postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create currency adapter: %w", err)
	}
	return service.New(adapter), nil
}
