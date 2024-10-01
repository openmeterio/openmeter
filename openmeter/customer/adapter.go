package customer

import (
	"context"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	CustomerAdapter

	DB() entcontext.DB
}

type CustomerAdapter interface {
	Register(observer appobserver.Observer[customerentity.Customer]) error
	Deregister(observer appobserver.Observer[customerentity.Customer]) error

	ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error)
	CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error)
	DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error)
	UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error)
}