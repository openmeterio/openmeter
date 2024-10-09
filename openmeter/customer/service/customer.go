package customerservice

import (
	"context"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[customerentity.Customer]{}, customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.ListCustomers(ctx, input)
}

func (s *Service) CreateCustomer(ctx context.Context, input customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.CreateCustomer(ctx, input)
}

func (s *Service) DeleteCustomer(ctx context.Context, input customerentity.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.DeleteCustomer(ctx, input)
}

func (s *Service) GetCustomer(ctx context.Context, input customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetCustomer(ctx, input)
}

func (s *Service) UpdateCustomer(ctx context.Context, input customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	if err := input.Validate(); err != nil {
		return nil, customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.UpdateCustomer(ctx, input)
}

func (s *Service) UpsertAppCustomer(ctx context.Context, input customerentity.UpsertAppCustomerInput) error {
	if err := input.Validate(); err != nil {
		return customerentity.ValidationError{
			Err: err,
		}
	}

	return s.adapter.UpsertAppCustomer(ctx, input)
}
