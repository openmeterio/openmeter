package appservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

var _ appstripe.AppService = (*Service)(nil)

func (s *Service) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.StripeCheckoutSession{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create checkout session: %w", err),
		}
	}

	return s.adapter.CreateCheckoutSession(ctx, input)
}
