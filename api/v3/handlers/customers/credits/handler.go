package customerscredits

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type customerBalanceFacade interface {
	GetBalances(ctx context.Context, input customerbalance.GetBalancesInput) ([]customerbalance.BalanceByCurrency, error)
}

type Handler interface {
	GetCustomerCreditBalance() GetCustomerCreditBalanceHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	customerService  customer.Service
	balanceFacade    customerBalanceFacade
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	customerService customer.Service,
	balanceFacade customerBalanceFacade,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		customerService:  customerService,
		balanceFacade:    balanceFacade,
		options:          options,
	}
}
