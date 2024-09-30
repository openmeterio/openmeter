package customer

import (
	"context"
	"errors"

	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CustomerService
}

type CustomerService interface {
	Register(observer appobserver.Observer[customerentity.Customer]) error
	Deregister(observer appobserver.Observer[customerentity.Customer]) error

	ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error)
	CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error)
	DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error)
	UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error)
}

type service struct {
	adapter Adapter
}

type ServiceConfig struct {
	Adapter Adapter
}

func (c *ServiceConfig) Validate() error {
	if c.Adapter == nil {
		return errors.New("adapter is required")
	}

	return nil
}

func NewService(c ServiceConfig) (Service, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &service{
		adapter: c.Adapter,
	}, nil
}

func (s *service) Register(observer appobserver.Observer[customerentity.Customer]) error {
	return s.adapter.Register(observer)
}

func (s *service) Deregister(observer appobserver.Observer[customerentity.Customer]) error {
	return s.adapter.Deregister(observer)
}

func (s *service) ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return s.adapter.ListCustomers(ctx, params)
}

func (s *service) CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return WithTx(ctx, s.adapter, func(ctx context.Context, adapter TxAdapter) (*customerentity.Customer, error) {
		return adapter.CreateCustomer(ctx, params)
	})
}

func (s *service) DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error {
	return WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter TxAdapter) error {
		return adapter.DeleteCustomer(ctx, customer)
	})
}

func (s *service) GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return s.adapter.GetCustomer(ctx, customer)
}

func (s *service) UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return WithTx(ctx, s.adapter, func(ctx context.Context, adapter TxAdapter) (*customerentity.Customer, error) {
		return adapter.UpdateCustomer(ctx, params)
	})
}
