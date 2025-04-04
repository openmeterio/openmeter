package customer

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CustomerService
	RequestValidatorService
}

type RequestValidatorService interface {
	RegisterRequestValidator(RequestValidator)
}

type CustomerService interface {
	ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.PagedResponse[Customer], error)
	CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, customer DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer GetCustomerInput) (*Customer, error)
	UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error)
	CustomerExists(ctx context.Context, customer CustomerID) error

	GetEntitlementValue(ctx context.Context, input GetEntitlementValueInput) (entitlement.EntitlementValue, error)
	GetCustomerAccess(ctx context.Context, input GetCustomerInput) (entitlement.Access, error)
}
