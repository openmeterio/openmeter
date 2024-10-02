package appservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.App{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create stripe app: %w", err),
		}
	}

	return s.adapter.CreateStripeApp(ctx, input)
}

func (s *Service) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error upsert stripe customer data: %w", err),
		}
	}

	return s.adapter.UpsertStripeCustomerData(ctx, input)
}

func (s *Service) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error delete stripe customer data: %w", err),
		}
	}

	return s.adapter.DeleteStripeCustomerData(ctx, input)
}
