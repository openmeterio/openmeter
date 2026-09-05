package customerbalance

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
)

type noopBalance struct{}

func (noopBalance) Settled() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

func (noopBalance) Live() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

func (noopBalance) Pending() alpacadecimal.Decimal {
	return alpacadecimal.Zero
}

type NoopService struct{}

var _ Service = NoopService{}

func (NoopService) GetBalance(context.Context, GetBalanceServiceInput) (Balance, error) {
	return noopBalance{}, nil
}

func (NoopService) GetSettledBalance(context.Context, GetBalanceServiceInput) (alpacadecimal.Decimal, error) {
	return alpacadecimal.Zero, nil
}

func (NoopService) ListCreditTransactions(context.Context, ListCreditTransactionsInput) (ListCreditTransactionsResult, error) {
	return ListCreditTransactionsResult{}, nil
}

func (NoopService) GetBalanceCurrencies(_ context.Context, input GetBalanceCurrenciesInput) ([]currencies.CurrencyReference, error) {
	references := make([]currencies.CurrencyReference, 0, len(input.Currencies.Codes))
	for _, code := range dedupeCurrencies(input.Currencies.Codes) {
		if code.IsFiat() {
			references = append(references, currencies.NewCurrencyReference(code))
		}
	}

	return references, nil
}

func NewNoopService() Service {
	return NoopService{}
}
