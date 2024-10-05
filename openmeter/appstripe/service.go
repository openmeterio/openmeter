package appstripe

import (
	"context"

	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

type Service interface {
	AppService
}

type AppService interface {
	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.StripeCheckoutSession, error)
}
