package customer

import (
	"context"
	"errors"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	apputils "github.com/openmeterio/openmeter/openmeter/app/utils"
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
	repo        Repository
	integration apputils.IntegrationGetter[Integration]
}

type ServiceConfig struct {
	Repository Repository
	AppGetter  apputils.AppGetter
}

func (c *ServiceConfig) Validate() error {
	if c.Repository == nil {
		return errors.New("repository is required")
	}

	if c.AppGetter == nil {
		return errors.New("app getter is required")
	}

	return nil
}

func NewService(c ServiceConfig) (Service, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &service{
		repo: c.Repository,
		integration: apputils.IntegrationGetter[Integration]{
			Getter: c.AppGetter,
		},
	}, nil
}

func (s *service) ListCustomers(ctx context.Context, params ListCustomersInput) (pagination.PagedResponse[Customer], error) {
	return s.repo.ListCustomers(ctx, params)
}

func (s *service) CreateCustomer(ctx context.Context, params CreateCustomerInput) (*Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*Customer, error) {
		return repo.CreateCustomer(ctx, params)
	})
}

func (s *service) DeleteCustomer(ctx context.Context, customer DeleteCustomerInput) error {
	return WithTxNoValue(ctx, s.repo, func(ctx context.Context, repo TxRepository) error {
		return repo.DeleteCustomer(ctx, customer)
	})
}

func (s *service) GetCustomer(ctx context.Context, customer GetCustomerInput) (*Customer, error) {
	return s.repo.GetCustomer(ctx, customer)
}

func (s *service) UpdateCustomer(ctx context.Context, params UpdateCustomerInput) (*Customer, error) {
	return WithTx(ctx, s.repo, func(ctx context.Context, repo TxRepository) (*Customer, error) {
		updatedCustomer, err := repo.UpdateCustomer(ctx, params)
		if err != nil {
			return nil, err
		}

		for _, appID := range updatedCustomer.AppIDs {
			integration, err := s.integration.Get(ctx, appentity.AppID{
				Namespace: updatedCustomer.Namespace,
				ID:        appID,
			})
			if err != nil {
				return nil, err
			}

			if err := integration.ValidateCustomer(ctx, *updatedCustomer); err != nil {
				return nil, err
			}
		}

		return updatedCustomer, nil
	})
}
