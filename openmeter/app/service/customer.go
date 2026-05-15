package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.Result[app.CustomerApp], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[app.CustomerApp]{}, models.NewGenericValidationError(err)
	}

	return s.adapter.ListCustomerData(ctx, input)
}

func (s *Service) EnsureCustomer(ctx context.Context, input app.EnsureCustomerInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.EnsureCustomer(ctx, input)
	})
}

func (s *Service) DeleteCustomer(ctx context.Context, input app.DeleteCustomerInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteCustomer(ctx, input)
	})
}
