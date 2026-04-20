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

var _ Service = NoopService{}

func (NoopService) GetBalance(context.Context, customer.CustomerID, currencyx.Code, *ledger.TransactionCursor) (ledger.Balance, error) {
	return noopBalance{}, nil
}

func (NoopService) ListCreditTransactions(context.Context, ListCreditTransactionsInput) (ListCreditTransactionsResult, error) {
	return ListCreditTransactionsResult{}, nil
}

func (NoopService) GetFBOCurrencies(context.Context, customer.CustomerID) ([]currencyx.Code, error) {
	return nil, nil
}

func NewNoopService() Service {
	return NoopService{}
}
