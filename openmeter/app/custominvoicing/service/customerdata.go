package service

import (
	"context"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *Service) GetCustomerData(ctx context.Context, input appcustominvoicing.GetAppCustomerDataInput) (appcustominvoicing.CustomerData, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (appcustominvoicing.CustomerData, error) {
		return s.adapter.GetCustomerData(ctx, input)
	})
}

func (s *Service) UpsertCustomerData(ctx context.Context, input appcustominvoicing.UpsertCustomerDataInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.UpsertCustomerData(ctx, input)
	})
}

func (s *Service) DeleteCustomerData(ctx context.Context, input appcustominvoicing.DeleteAppCustomerDataInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteCustomerData(ctx, input)
	})
}
