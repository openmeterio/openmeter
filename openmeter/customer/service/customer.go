package customerservice

import (
	"context"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

func (s *Service) Register(observer appobserver.Observer[customerentity.Customer]) error {
	return s.adapter.Register(observer)
}

func (s *Service) Deregister(observer appobserver.Observer[customerentity.Customer]) error {
	return s.adapter.Deregister(observer)
}

func (s *Service) ListCustomers(ctx context.Context, input customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return entcontext.WithTx(ctx, s.adapter.DB(), func(ctx context.Context) (*customerentity.Customer, error) {
		return s.adapter.CreateCustomer(ctx, input)
	})
}

func (s *Service) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	return entcontext.WithTxNoValue(ctx, s.adapter.DB(), func(ctx context.Context) error {
		return s.adapter.DeleteCustomer(ctx, input)
	})
}

func (s *Service) GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.GetCustomer(ctx, customer)
}

func (s *Service) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return entcontext.WithTx(ctx, s.adapter.DB(), func(ctx context.Context) (*customerentity.Customer, error) {
		return s.adapter.UpdateCustomer(ctx, input)
	})
}
