package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
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

// CreateCheckoutSession creates a new checkout session
func (a adapter) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.StripeCheckoutSession, error) {
	// Get the stripe app
	stripeApp, err := a.db.AppStripe.
		Query().
		Where(appstripedb.ID(input.AppID.ID)).
		Where(appstripedb.Namespace(input.AppID.Namespace)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.StripeCheckoutSession{}, app.AppNotFoundError{
				AppID: input.AppID,
			}
		}

		return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get the stripe app customer
	var stripeCustomerId string
	{
		stripeAppCustomer, err := a.db.AppStripeCustomer.
			Query().
			Where(appstripecustomerdb.AppID(input.AppID.ID)).
			Where(appstripecustomerdb.Namespace(input.AppID.Namespace)).
			Where(appstripecustomerdb.CustomerID(input.CustomerID.ID)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				// If Stripe Customer ID is provided we need to upsert it
				if input.StripeCustomerID != nil {
					err = a.UpsertStripeCustomerData(ctx, appstripeentity.UpsertStripeCustomerDataInput{
						AppID:            input.AppID,
						CustomerID:       input.CustomerID,
						StripeCustomerID: *input.StripeCustomerID,
					})
					if err != nil {
						return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to upsert stripe customer data: %w", err)
					}

					stripeCustomerId = *input.StripeCustomerID
				} else {
					// Otherwise we create a new Stripe Customer
					out, err := a.CreateStripeCustomer(ctx, appstripeentity.CreateStripeCustomerInput{
						AppID:      input.AppID,
						CustomerID: input.CustomerID,
					})
					if err != nil {
						return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to create stripe customer: %w", err)
					}

					stripeCustomerId = out.StripeCustomerID
				}
			}

			return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to get stripe app customer: %w", err)
		}

		// If the stripe app customer exists we check if the Stripe Customer ID matches with the input
		if stripeAppCustomer != nil {
			if input.StripeCustomerID != nil && input.StripeCustomerID != stripeAppCustomer.StripeCustomerID {
				return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("stripe customer id mismatch the one stored for customer: %s != %s", *input.StripeCustomerID, *stripeAppCustomer.StripeCustomerID)
			}

			stripeCustomerId = *stripeAppCustomer.StripeCustomerID
		}
	}

	// We set the Stripe Customer ID
	input.StripeCustomerID = &stripeCustomerId

	// Get Stripe API Key
	apiKeySecret, err := a.secretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.ID,
		},
		Key: *stripeApp.APIKey,
	})
	if err != nil {
		return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.stripeClientFactory(appstripeentity.StripeClientConfig{
		Namespace: stripeApp.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Create the checkout session
	checkoutSession, err := stripeClient.CreateCheckoutSession(ctx, appstripeentity.StripeClientCreateCheckoutSessionInput{
		StripeCustomerID: stripeCustomerId,
		AppID:            input.AppID,
		CustomerID:       input.CustomerID,
		Options:          input.Options,
	})
	if err != nil {
		return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to create checkout session: %w", err)
	}

	if err := checkoutSession.Validate(); err != nil {
		return appstripeentity.StripeCheckoutSession{}, fmt.Errorf("failed to validate checkout session: %w", err)
	}

	return checkoutSession, nil
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
