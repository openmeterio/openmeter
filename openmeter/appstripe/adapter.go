package appstripe

import (
	"context"

	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	entcontext "github.com/openmeterio/openmeter/pkg/framework/entutils/context"
)

type Adapter interface {
	AppStripeAdapter

	DB() entcontext.DB
}

type AppStripeAdapter interface {
	CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error)
	UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error
	DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error
}
