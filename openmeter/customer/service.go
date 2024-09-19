package customer

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CustomerService

	Close() error
}

type CustomerService interface {
	ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.PagedResponse[Customer], error)
	CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, params DeleteCustomerInput) error
	GetCustomer(ctx context.Context, params GetCustomerInput) (*Customer, error)
	UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error)
}
