package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.App{}, appstripe.ValidationError{
			Err: err,
		}
	}

	return entcontext.WithTx(ctx, s.adapter.DB(), func(ctx context.Context) (appstripeentity.App, error) {
		return s.adapter.CreateStripeApp(ctx, input)
	})
}

func (s *Service) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: err,
		}
	}

	return s.adapter.UpsertStripeCustomerData(ctx, input)
}

func (s *Service) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: err,
		}
	}

	return entcontext.WithTxNoValue(ctx, s.adapter.DB(), func(ctx context.Context) error {
		return s.adapter.DeleteStripeCustomerData(ctx, input)
	})
}
