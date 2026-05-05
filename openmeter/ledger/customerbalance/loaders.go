package customerbalance

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type creditTransactionLoaderInput struct {
	Limit      int
	After      *ledger.TransactionCursor
	Before     *ledger.TransactionCursor
	CustomerID customer.CustomerID
	AccountID  string
	Currencies []currencyx.Code
}

type creditTransactionLoaderResult struct {
	Items   []CreditTransaction
	HasMore bool
}

type creditTransactionLoader interface {
	Load(ctx context.Context, input creditTransactionLoaderInput) (creditTransactionLoaderResult, error)
}

type creditTransactionLoaderFactory func(*service) creditTransactionLoader

var creditTransactionLoaderOrder = []CreditTransactionType{
	CreditTransactionTypeFunded,
	CreditTransactionTypeConsumed,
}

var creditTransactionLoaderFactories = map[CreditTransactionType]creditTransactionLoaderFactory{
	CreditTransactionTypeFunded: newFundedCreditTransactionLoader,
	CreditTransactionTypeConsumed: func(s *service) creditTransactionLoader {
		return newLedgerCreditTransactionLoader(s, ledger.ListTransactionsCreditMovementNegative)
	},
}

func (s *service) creditTransactionLoaders(txTypes []CreditTransactionType) ([]creditTransactionLoader, error) {
	if len(txTypes) == 0 {
		loaders := make([]creditTransactionLoader, 0, len(creditTransactionLoaderOrder))
		for _, transactionType := range creditTransactionLoaderOrder {
			loaders = append(loaders, creditTransactionLoaderFactories[transactionType](s))
		}

		return loaders, nil
	}

	loaders := make([]creditTransactionLoader, 0, len(txTypes))
	for _, txType := range txTypes {
		factory, ok := creditTransactionLoaderFactories[txType]
		if !ok {
			return nil, txType.Validate()
		}

		loaders = append(loaders, factory(s))
	}

	return loaders, nil
}
