package customerbalance

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type noopBalance struct{}

func (noopBalance) Settled() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

func (noopBalance) Pending() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

type NoopService struct{}

var _ FacadeService = NoopService{}

func (NoopService) GetBalance(context.Context, customer.CustomerID, ledger.RouteFilter) (ledger.Balance, error) {
	return noopBalance{}, nil
}

func (NoopService) getFBOCurrencies(context.Context, customer.CustomerID) ([]currencyx.Code, error) {
	return nil, nil
}

func NewNoopService() FacadeService {
	return NoopService{}
}
