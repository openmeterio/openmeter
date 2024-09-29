package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.App{}, appstripe.ValidationError{
			Err: err,
		}
	}

	return appstripe.WithTx(ctx, s.adapter, func(ctx context.Context, adapter appstripe.TxAdapter) (appstripeentity.App, error) {
		return adapter.CreateStripeApp(ctx, input)
	})
}

func (s *Service) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: err,
		}
	}

	return appstripe.WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter appstripe.TxAdapter) error {
		return adapter.UpsertStripeCustomerData(ctx, input)
	})
}

func (s *Service) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: err,
		}
	}

	return appstripe.WithTxNoValue(ctx, s.adapter, func(ctx context.Context, adapter appstripe.TxAdapter) error {
		return adapter.DeleteStripeCustomerData(ctx, input)
	})
}
