package customerscredits

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type customerBalanceFacade interface {
	GetBalance(ctx context.Context, input customerbalance.GetBalanceInput) (alpacadecimal.Decimal, error)
	GetBalances(ctx context.Context, input customerbalance.GetBalancesInput) ([]customerbalance.BalanceByCurrency, error)
	ListCreditTransactions(ctx context.Context, input customerbalance.ListCreditTransactionsInput) (customerbalance.ListCreditTransactionsResult, error)
}

type Handler interface {
	GetCustomerCreditBalance() GetCustomerCreditBalanceHandler
	ListCreditGrants() ListCreditGrantsHandler
	CreateCreditGrant() CreateCreditGrantHandler
	GetCreditGrant() GetCreditGrantHandler
	ListCreditTransactions() ListCreditTransactionsHandler
}

type handler struct {
	resolveNamespace   func(ctx context.Context) (string, error)
	customerService    customer.Service
	balanceFacade      customerBalanceFacade
	creditGrantService creditgrant.Service
	ledger             ledger.Ledger
	accountResolver    ledger.AccountResolver
	options            []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	balanceFacade customerBalanceFacade,
	creditGrantService creditgrant.Service,
	ledger ledger.Ledger,
	accountResolver ledger.AccountResolver,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace:   resolveNamespace,
		customerService:    customerService,
		balanceFacade:      balanceFacade,
		creditGrantService: creditGrantService,
		ledger:             ledger,
		accountResolver:    accountResolver,
		options:            options,
	}
}
