// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
