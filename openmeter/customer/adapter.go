package customer

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	CustomerAdapter

	entutils.TxCreator
}

type CustomerAdapter interface {
	ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.Result[Customer], error)
	CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, customer DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer GetCustomerInput) (*Customer, error)
	GetCustomerByUsageAttribution(ctx context.Context, input GetCustomerByUsageAttributionInput) (*Customer, error)
	UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error)
}
