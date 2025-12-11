package customers

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type Handler interface {
	ListCustomers() ListCustomersHandler
	//CreateCustomer() CreateCustomerHandler
	//DeleteCustomer() DeleteCustomerHandler
	//GetCustomer() GetCustomerHandler
	//UpdateCustomer() UpdateCustomerHandler
}

type handler struct {
	resolveNamespace func(ctx context.Context) (string, error)
	service          customer.Service
	options          []httptransport.HandlerOption
}

func New(
	resolveNamespace func(ctx context.Context) (string, error),
	service customer.Service,
	options ...httptransport.HandlerOption,
) Handler {
	return &handler{
		resolveNamespace: resolveNamespace,
		service:          service,
		options:          options,
	}
}
