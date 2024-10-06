package appstripe

import (
	"context"

	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

type Service interface {
	AppService
}

type AppService interface {
	CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error)
}
