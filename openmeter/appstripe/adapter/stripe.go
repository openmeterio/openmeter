package appstripeadapter

import (
	"context"
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/secret"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.App{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create stripe app: %w", err),
		}
	}

	return transaction.Run(ctx, a, func(ctx context.Context) (appstripeentity.App, error) {
		// Create the base app
		appBase, err := a.appService.CreateApp(ctx, appentity.CreateAppInput{
			Namespace:   input.Namespace,
			Name:        input.Name,
			Description: input.Description,
			Type:        appentitybase.AppTypeStripe,
		})
		if err != nil {
			return appstripeentity.App{}, fmt.Errorf("failed to create app: %w", err)
		}

		// Create the stripe app in the database
		appStripeCreateQuery := a.db.AppStripe.Create().
			SetID(appBase.GetID().ID).
			SetNamespace(input.Namespace).
			SetStripeAccountID(input.StripeAccountID).
			SetStripeLivemode(input.Livemode).
			SetAPIKey(input.APIKey.ID)

		dbAppStripe, err := appStripeCreateQuery.Save(ctx)
		if err != nil {
			return appstripeentity.App{}, fmt.Errorf("failed to create stripe app: %w", err)
		}

		// Map the database stripe app to an app entity
		app, err := mapAppStripeFromDB(appBase, dbAppStripe, a.db, a.secretService, a.stripeClientFactory)
		if err != nil {
			return appstripeentity.App{}, err
		}

		return app, nil
	})
}

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(
	appBase appentitybase.AppBase,
	dbAppStripe *db.AppStripe,
	client *entdb.Client,
	secretService secret.Service,
	stripeClientFactory appstripeentity.StripeClientFactory,
) (appstripeentity.App, error) {
	app := appstripeentity.App{
		AppBase:         appBase,
		Livemode:        dbAppStripe.StripeLivemode,
		StripeAccountId: dbAppStripe.StripeAccountID,

		Client:              client,
		SecretService:       secretService,
		StripeClientFactory: stripeClientFactory,
	}

	if err := app.Validate(); err != nil {
		return appstripeentity.App{}, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}
