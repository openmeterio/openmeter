package common

import (
	"github.com/google/wire"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/currencies/service"
)

var Currency = wire.NewSet(
	NewCurrencyService,
)

func NewCurrencyService() currencies.CurrencyService {
	return service.New()
}
