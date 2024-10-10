package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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
			SetAPIKey(input.APIKey.ID).
			SetWebhookSecret(input.WebhookSecret.ID)

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

// GetStripeApp gets an app
func (a adapter) GetStripeApp(ctx context.Context, input appstripeentity.GetAppInput) (appstripeentity.App, error) {
	app, err := a.appService.GetApp(ctx, input)
	if err != nil {
		return appstripeentity.App{}, err
	}

	if stripeApp, ok := app.(appstripeentity.App); ok {
		return stripeApp, nil
	}

	return appstripeentity.App{}, fmt.Errorf("app is not a stripe app")
}

// GetWebhookSecret gets the webhook secret
func (a adapter) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	if err := input.Validate(); err != nil {
		return secretentity.Secret{}, appstripe.ValidationError{
			Err: fmt.Errorf("error get webhook secret: %w", err),
		}
	}

	// Get the stripe app
	stripeApp, err := a.db.AppStripe.
		Query().
		Where(appstripedb.ID(input.AppID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return secretentity.Secret{}, appstripe.WebhookAppNotFoundError{
				AppID: input.AppID,
			}
		}

		return secretentity.Secret{}, fmt.Errorf("failed to get stripe app: %w", err)
	}

	appID := appentitybase.AppID{
		Namespace: stripeApp.Namespace,
		ID:        stripeApp.ID,
	}

	// Get the webhook secret
	secret, err := a.secretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.WebhookSecret,
		},
		AppID: appID,
		Key:   appstripeentity.WebhookSecretKey,
	})
	if err != nil {
		return secretentity.Secret{}, fmt.Errorf("failed to get webhook secret: %w", err)
	}

	return secret, nil
}

// SetCustomerDefaultPaymentMethod sets the default payment method for a customer
func (a adapter) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, appstripe.ValidationError{
			Err: fmt.Errorf("error set customer default payment method: %w", err),
		}
	}

	// Get the stripe app customer
	appCustomer, err := a.db.AppStripeCustomer.
		Query().
		Where(
			appstripecustomerdb.Namespace(input.AppID.Namespace),
			appstripecustomerdb.AppID(input.AppID.ID),
			appstripecustomerdb.StripeCustomerID(input.StripeCustomerID),
		).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, appstripe.StripeCustomerPreConditionError{
				AppID:            input.AppID,
				StripeCustomerID: input.StripeCustomerID,
				Condition:        "stripe customer has no data for stripe app",
			}
		}
	}

	customerID := customerentity.CustomerID{
		Namespace: input.AppID.Namespace,
		ID:        appCustomer.CustomerID,
	}

	// Should not happen as we filter in database query for stripe customer id
	if appCustomer.StripeCustomerID == nil {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, appstripe.StripeCustomerPreConditionError{
			AppID:            input.AppID,
			StripeCustomerID: input.StripeCustomerID,
			Condition:        "stripe customer id is not set",
		}
	}

	// Check if the stripe customer id matches with the input
	if *appCustomer.StripeCustomerID != input.StripeCustomerID {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, app.CustomerPreConditionError{
			AppID:      input.AppID,
			CustomerID: customerID,
			Condition:  "customer stripe customer id mismatch",
		}
	}

	// Set the default payment method
	return transaction.Run(ctx, a, func(ctx context.Context) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
		_, err := a.db.AppStripeCustomer.
			Update().
			Where(
				appstripecustomerdb.Namespace(input.AppID.Namespace),
				appstripecustomerdb.AppID(input.AppID.ID),
				appstripecustomerdb.CustomerID(customerID.ID),
			).
			SetStripeDefaultPaymentMethodID(input.PaymentMethodID).
			Save(ctx)
		if err != nil {
			return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, fmt.Errorf("failed to set customer default payment method: %w", err)
		}

		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{
			CustomerID: customerID,
		}, nil
	})
}

// CreateCheckoutSession creates a new checkout session
func (a adapter) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.CreateCheckoutSessionOutput{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create checkout session: %w", err),
		}
	}

	return transaction.Run(ctx, a, func(ctx context.Context) (appstripeentity.CreateCheckoutSessionOutput, error) {
		// Get the stripe app
		stripeApp, err := a.db.AppStripe.
			Query().
			Where(appstripedb.ID(input.AppID.ID)).
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return appstripeentity.CreateCheckoutSessionOutput{}, appstripe.AppNotFoundError{
					AppID: input.AppID,
				}
			}

			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe app: %w", err)
		}

		// Get customer
		// if input.CustomerID != nil {
		_, err = a.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(input.CustomerID))
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get customer: %w", err)
		}
		// }

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
							return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to upsert stripe customer data: %w", err)
						}

						stripeCustomerId = *input.StripeCustomerID
					} else {
						// Otherwise we create a new Stripe Customer
						out, err := a.CreateStripeCustomer(ctx, appstripeentity.CreateStripeCustomerInput{
							AppID:      input.AppID,
							CustomerID: input.CustomerID,
						})
						if err != nil {
							return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create stripe customer: %w", err)
						}

						stripeCustomerId = out.StripeCustomerID
					}
				} else {
					return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe app customer: %w", err)
				}
			}

			// If the stripe app customer exists we check if the Stripe Customer ID matches with the input
			if stripeAppCustomer != nil {
				if input.StripeCustomerID != nil && input.StripeCustomerID != stripeAppCustomer.StripeCustomerID {
					return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("stripe customer id mismatch the one stored for customer: %s != %s", *input.StripeCustomerID, *stripeAppCustomer.StripeCustomerID)
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
				ID:        stripeApp.APIKey,
			},
			AppID: input.AppID,
			Key:   appstripeentity.APIKeySecretKey,
		})
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
		}

		// Stripe Client
		stripeClient, err := a.stripeClientFactory(stripeclient.StripeClientConfig{
			Namespace: stripeApp.Namespace,
			APIKey:    apiKeySecret.Value,
		})
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create stripe client: %w", err)
		}

		// Create the checkout session
		checkoutSession, err := stripeClient.CreateCheckoutSession(ctx, stripeclient.CreateCheckoutSessionInput{
			StripeCustomerID: stripeCustomerId,
			AppID:            input.AppID,
			CustomerID:       input.CustomerID,
			Options:          input.Options,
		})
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create checkout session: %w", err)
		}

		if err := checkoutSession.Validate(); err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to validate checkout session: %w", err)
		}

		return appstripeentity.CreateCheckoutSessionOutput{
			CustomerID:       input.CustomerID,
			StripeCustomerID: stripeCustomerId,

			SessionID:     checkoutSession.SessionID,
			SetupIntentID: checkoutSession.SetupIntentID,
			URL:           checkoutSession.URL,
			Mode:          checkoutSession.Mode,
			CancelURL:     checkoutSession.CancelURL,
			SuccessURL:    checkoutSession.SuccessURL,
			ReturnURL:     checkoutSession.ReturnURL,
		}, nil
	})
}

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(
	appBase appentitybase.AppBase,
	dbAppStripe *db.AppStripe,
	client *entdb.Client,
	secretService secret.Service,
	stripeClientFactory stripeclient.StripeClientFactory,
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
