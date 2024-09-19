package customer

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	CustomerService
}

type CustomerService interface {
	ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.PagedResponse[Customer], error)
	CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, customer DeleteCustomerInput) error
	GetCustomer(ctx context.Context, customer GetCustomerInput) (*Customer, error)
	UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error)
}

type service struct {
	repo Repository
}

type Config struct {
	Repository Repository
}

func (c *Config) Validate() error {
	if c.Repository == nil {
		return errors.New("repository is required")
	}

	return nil
}

func NewService(c Config) (Service, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &service{
		repo: c.Repository,
	}, nil
}

func (s *service) ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.PagedResponse[Customer], error) {
	return s.repo.ListCustomers(ctx, params)
}

func (s *service) CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*Customer, error) {
		return s.repo.CreateCustomer(ctx, params)
	})
}

func (s *service) DeleteCustomer(ctx context.Context, customer DeleteCustomerInput) error {
	return WithTxNoValue(ctx, s.repo, func(ctx context.Context, repo TxRepository) error {
		return s.repo.DeleteCustomer(ctx, customer)
	})
}

func (s *service) GetCustomer(ctx context.Context, customer GetCustomerInput) (*Customer, error) {
	return s.repo.GetCustomer(ctx, customer)
}

func (s *service) UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*Customer, error) {
		return repo.UpdateCustomer(ctx, params)
	})
}
