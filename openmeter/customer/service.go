package customer

import (
	"context"
	"errors"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CustomerService
}

type CustomerService interface {
	ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error)
	CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error)
	DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error)
	UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error)
}

type service struct {
	repo Repository
}

type ServiceConfig struct {
	Repository Repository
}

func (c *ServiceConfig) Validate() error {
	if c.Repository == nil {
		return errors.New("repository is required")
	}

	return nil
}

func NewService(c ServiceConfig) (Service, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &service{
		repo: c.Repository,
	}, nil
}

func (s *service) ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return s.repo.ListCustomers(ctx, params)
}

func (s *service) CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*customerentity.Customer, error) {
		return repo.CreateCustomer(ctx, params)
	})
}

func (s *service) DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error {
	return WithTxNoValue(ctx, s.repo, func(ctx context.Context, repo TxRepository) error {
		return repo.DeleteCustomer(ctx, customer)
	})
}

func (s *service) GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return s.repo.GetCustomer(ctx, customer)
}

func (s *service) UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*customerentity.Customer, error) {
		return repo.UpdateCustomer(ctx, params)
	})
}
