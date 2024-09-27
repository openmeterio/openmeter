package appstripe

import (
	"context"

	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
)

type Service interface {
	AppService
}

type AppService interface {
	CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error)
}
