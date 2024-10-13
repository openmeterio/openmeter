package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentityapp.App, error) {
	if err := input.Validate(); err != nil {
		return appstripeentityapp.App{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create stripe app: %w", err),
		}
	}

	return transaction.Run(ctx, a, func(ctx context.Context) (appstripeentityapp.App, error) {
		// Create the base app
		appBase, err := a.appService.CreateApp(ctx, appentity.CreateAppInput{
			Namespace:   input.Namespace,
			Name:        input.Name,
			Description: input.Description,
			Type:        appentitybase.AppTypeStripe,
		})
		if err != nil {
			return appstripeentityapp.App{}, fmt.Errorf("failed to create app: %w", err)
		}

		// Create the stripe app in the database
		appStripeCreateQuery := a.db.AppStripe.Create().
			SetID(appBase.GetID().ID).
			SetNamespace(input.Namespace).
			SetStripeAccountID(input.StripeAccountID).
			SetStripeLivemode(input.Livemode).
			SetAPIKey(input.APIKey.ID).
			SetStripeWebhookID(input.StripeWebhookID).
			SetWebhookSecret(input.WebhookSecret.ID)

		dbApp, err := appStripeCreateQuery.Save(ctx)
		if err != nil {
			return appstripeentityapp.App{}, fmt.Errorf("failed to create stripe app: %w", err)
		}

		// Map the database stripe app to an app entity
		appData := mapAppStripeData(appBase.GetID(), dbApp)

		// Map the database stripe app to an app entity
		app, err := a.mapAppStripeFromDB(appBase, appData)
		if err != nil {
			return appstripeentityapp.App{}, err
		}

		return app, nil
	})
}

// GetStripeAppData gets stripe customer data
func (a adapter) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.AppData{}, appstripe.ValidationError{
			Err: fmt.Errorf("error getting stripe customer data: %w", err),
		}
	}

	dbApp, err := a.db.AppStripe.
		Query().
		Where(appstripedb.Namespace(input.AppID.Namespace)).
		Where(appstripedb.ID(input.AppID.ID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.AppData{}, app.AppNotFoundError{
				AppID: input.AppID,
			}
		}

		return appstripeentity.AppData{}, fmt.Errorf("error getting stripe customer data: %w", err)
	}

	// Map the database stripe app to an app entity
	appData := mapAppStripeData(input.AppID, dbApp)
	if err := appData.Validate(); err != nil {
		return appstripeentity.AppData{}, fmt.Errorf("error validating stripe app data: %w", err)
	}

	return appData, nil
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
		// We intentionally do not filter by namespace as the webhook payload is signed with the secret
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

	// Get the webhook secret
	secret, err := a.secretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.WebhookSecret,
		},
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

	// Check if the stripe customer id matches with the input
	if appCustomer.StripeCustomerID != input.StripeCustomerID {
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
		var appID appentitybase.AppID

		// Use the provided app ID or get the default Stripe app
		if input.AppID != nil {
			appID = *input.AppID
		} else {
			app, err := a.appService.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
				Namespace: input.Namespace,
				Type:      appentitybase.AppTypeStripe,
			})
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get default app: %w", err)
			}

			appID = app.GetID()
		}

		// Get the stripe app
		stripeApp, err := a.db.AppStripe.
			Query().
			Where(appstripedb.ID(appID.ID)).
			Where(appstripedb.Namespace(appID.Namespace)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return appstripeentity.CreateCheckoutSessionOutput{}, appstripe.AppNotFoundError{
					AppID: appID,
				}
			}

			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe app: %w", err)
		}

		// Get or create customer
		var customer *customerentity.Customer

		if input.CustomerID != nil {
			customer, err = a.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(*input.CustomerID))
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get customer: %w", err)
			}
		}

		if input.CreateCustomerInput != nil {
			customer, err = a.customerService.CreateCustomer(ctx, *input.CreateCustomerInput)
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create customer: %w", err)
			}
		}

		customerID := customer.GetID()

		// Get the stripe app customer
		var stripeCustomerId string
		{
			stripeAppCustomer, err := a.db.AppStripeCustomer.
				Query().
				Where(appstripecustomerdb.AppID(appID.ID)).
				Where(appstripecustomerdb.Namespace(appID.Namespace)).
				Where(appstripecustomerdb.CustomerID(customerID.ID)).
				Only(ctx)
			if err != nil {
				if entdb.IsNotFound(err) {
					// If Stripe Customer ID is provided we need to upsert it
					if input.StripeCustomerID != nil {
						err = a.UpsertStripeCustomerData(ctx, appstripeentity.UpsertStripeCustomerDataInput{
							AppID:            appID,
							CustomerID:       customerID,
							StripeCustomerID: *input.StripeCustomerID,
						})
						if err != nil {
							return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to upsert stripe customer data: %w", err)
						}

						stripeCustomerId = *input.StripeCustomerID
					} else {
						// Otherwise we create a new Stripe Customer
						params := appstripeentity.CreateStripeCustomerInput{
							AppID:      appID,
							CustomerID: customerID,
							Name:       &customer.Name,
						}

						out, err := a.createStripeCustomer(ctx, params)
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
				if input.StripeCustomerID != nil && *input.StripeCustomerID != stripeAppCustomer.StripeCustomerID {
					return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("stripe customer id mismatch the one stored for customer: %s != %s", *input.StripeCustomerID, stripeAppCustomer.StripeCustomerID)
				}

				stripeCustomerId = stripeAppCustomer.StripeCustomerID
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
			AppID:            appID,
			CustomerID:       customerID,
			Options:          input.Options,
		})
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create checkout session: %w", err)
		}

		if err := checkoutSession.Validate(); err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to validate checkout session: %w", err)
		}

		return appstripeentity.CreateCheckoutSessionOutput{
			CustomerID:       customerID,
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
func (a adapter) mapAppStripeFromDB(
	appBase appentitybase.AppBase,
	stripeApp appstripeentity.AppData,
) (appstripeentityapp.App, error) {
	app := appstripeentityapp.App{
		AppBase: appBase,
		AppData: stripeApp,

		// TODO: fixme, it should be a service not an adapter
		// But the factory (this) is is in the adapter that the service depends on
		StripeAppService:    a,
		SecretService:       a.secretService,
		StripeClientFactory: a.stripeClientFactory,
	}

	if err := app.Validate(); err != nil {
		return appstripeentityapp.App{}, fmt.Errorf("failed to map stripe app from db: %w", err)
	}

	return app, nil
}

// mapAppStripeData maps stripe app data from the database
func mapAppStripeData(appID appentitybase.AppID, dbApp *entdb.AppStripe) appstripeentity.AppData {
	return appstripeentity.AppData{
		StripeAccountID: dbApp.StripeAccountID,
		Livemode:        dbApp.StripeLivemode,
		APIKey:          secretentity.NewSecretID(appID, dbApp.APIKey, appstripeentity.APIKeySecretKey),
		StripeWebhookID: dbApp.StripeWebhookID,
		WebhookSecret:   secretentity.NewSecretID(appID, dbApp.WebhookSecret, appstripeentity.WebhookSecretKey),
	}
}
