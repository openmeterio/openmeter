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

	return s.adapter.CreateStripeApp(ctx, input)
}