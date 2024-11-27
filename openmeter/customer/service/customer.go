package customerservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ customer.Service = (*Service)(nil)

func (s *Service) ListCustomers(ctx context.Context, input customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.CreateCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	return s.adapter.DeleteCustomer(ctx, input)
}

func (s *Service) GetCustomer(ctx context.Context, input customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.GetCustomer(ctx, input)
}

func (s *Service) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.UpdateCustomer(ctx, input)
}
