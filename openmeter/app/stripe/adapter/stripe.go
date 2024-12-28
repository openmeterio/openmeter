package appstripeadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// GetStripeClientFactory gets the stripe client factory
func (a adapter) GetStripeClientFactory() stripeclient.StripeClientFactory {
	return a.stripeClientFactory
}

// GetStripeAppClientFactory gets the stripe client factory
func (a adapter) GetStripeAppClientFactory() stripeclient.StripeAppClientFactory {
	return a.stripeAppClientFactory
}

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.AppBase, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.AppBase{}, appstripe.ValidationError{
			Err: fmt.Errorf("error create stripe app: %w", err),
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.AppBase, error) {
		// Create the base app
		appBase, err := repo.appService.CreateApp(ctx, appentity.CreateAppInput{
			Namespace:   input.Namespace,
			Name:        input.Name,
			Description: input.Description,
			Type:        appentitybase.AppTypeStripe,
		})
		if err != nil {
			return appstripeentity.AppBase{}, fmt.Errorf("failed to create app: %w", err)
		}

		// Create the stripe app in the database
		appStripeCreateQuery := repo.db.AppStripe.Create().
			SetID(appBase.GetID().ID).
			SetNamespace(input.Namespace).
			SetStripeAccountID(input.StripeAccountID).
			SetStripeLivemode(input.Livemode).
			SetAPIKey(input.APIKey.ID).
			SetStripeWebhookID(input.StripeWebhookID).
			SetWebhookSecret(input.WebhookSecret.ID)

		dbApp, err := appStripeCreateQuery.Save(ctx)
		if err != nil {
			return appstripeentity.AppBase{}, fmt.Errorf("failed to create stripe app: %w", err)
		}

		// Map the database stripe app to an app entity
		appData := mapAppStripeData(appBase.GetID(), dbApp)

		return appstripeentity.AppBase{
			AppBase: appBase,
			AppData: appData,
		}, nil
	})
}

// UpdateAPIKey replaces the API key
func (a adapter) UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error {
	// Validate the input
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error replace api key: %w", err),
		}
	}

	// Get the stripe app data
	appData, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: input.AppID,
	})
	if err != nil {
		return fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Validate the new API key
	stripeClient, err := a.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
		AppID:      input.AppID,
		AppService: a.appService,
		APIKey:     input.APIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Check if new API Key in the same live or test mode as the app
	livemode := stripeclient.IsAPIKeyLiveMode(input.APIKey)
	if livemode != appData.Livemode {
		var err error

		if livemode {
			err = errors.New("new stripe api key is in live mode but the app is in test mode")
		} else {
			err = errors.New("new stripe api key is in test mode but the app is in live mode")
		}

		return appstripe.ValidationError{
			Err: err,
		}
	}

	// Check if it belongs to the same stripe account
	stripeAccount, err := stripeClient.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate stripe api key: %w", err)
	}

	// Check if the stripe account id matches with the stored one
	if stripeAccount.StripeAccountID != appData.StripeAccountID {
		return appstripe.ValidationError{
			Err: fmt.Errorf("stripe account id mismatch: %s != %s", stripeAccount.StripeAccountID, appData.StripeAccountID),
		}
	}

	// Update the API key
	err = a.secretService.UpdateAppSecret(ctx, secretentity.UpdateAppSecretInput{
		AppID:    input.AppID,
		SecretID: appData.APIKey,
		Key:      appstripeentity.APIKeySecretKey,
		Value:    input.APIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to update api key: %w", err)
	}

	// Update the app status to ready
	status := appentitybase.AppStatusReady

	err = a.appService.UpdateAppStatus(ctx, appentity.UpdateAppStatusInput{
		ID:     input.AppID,
		Status: status,
	})
	if err != nil {
		return fmt.Errorf("failed to update app status to %s for %s: %w", input.AppID.ID, status, err)
	}

	return nil
}

// GetStripeAppData gets stripe customer data
func (a adapter) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.AppData{}, appstripe.ValidationError{
			Err: fmt.Errorf("error getting stripe customer data: %w", err),
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.AppData, error) {
		dbApp, err := repo.db.AppStripe.
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
	})
}

// GetWebhookSecret gets the webhook secret
func (a adapter) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	if err := input.Validate(); err != nil {
		return secretentity.Secret{}, appstripe.ValidationError{
			Err: fmt.Errorf("error get webhook secret: %w", err),
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.GetWebhookSecretOutput, error) {
		// Get the stripe app
		stripeApp, err := repo.db.AppStripe.
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
		appID := appentitybase.AppID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.ID,
		}

		secret, err := a.secretService.GetAppSecret(ctx, secretentity.NewSecretID(appID, stripeApp.WebhookSecret, appstripeentity.WebhookSecretKey))
		if err != nil {
			return secretentity.Secret{}, fmt.Errorf("failed to get webhook secret: %w", err)
		}

		return secret, nil
	})
}

// SetCustomerDefaultPaymentMethod sets the default payment method for a customer
func (a adapter) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, appstripe.ValidationError{
			Err: fmt.Errorf("error set customer default payment method: %w", err),
		}
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
		// Get the stripe app customer
		appCustomer, err := repo.db.AppStripeCustomer.
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

		_, err = repo.db.AppStripeCustomer.
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

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.CreateCheckoutSessionOutput, error) {
		var appID appentitybase.AppID

		// Use the provided app ID or get the default Stripe app
		if input.AppID != nil {
			appID = *input.AppID
		} else {
			app, err := repo.appService.GetDefaultApp(ctx, appentity.GetDefaultAppInput{
				Namespace: input.Namespace,
				Type:      appentitybase.AppTypeStripe,
			})
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get default app: %w", err)
			}

			appID = app.GetID()
		}

		// Get the stripe app
		stripeApp, err := repo.db.AppStripe.
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
			customer, err = repo.customerService.GetCustomer(ctx, customerentity.GetCustomerInput(*input.CustomerID))
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get customer: %w", err)
			}
		}

		if input.CreateCustomerInput != nil {
			customer, err = repo.customerService.CreateCustomer(ctx, *input.CreateCustomerInput)
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create customer: %w", err)
			}
		}

		customerID := customer.GetID()

		// Get the stripe app customer
		var stripeCustomerId string
		{
			stripeAppCustomer, err := repo.db.AppStripeCustomer.
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
		apiKeySecret, err := repo.secretService.GetAppSecret(ctx, secretentity.NewSecretID(appID, stripeApp.APIKey, appstripeentity.APIKeySecretKey))
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
		}

		// Stripe Client
		stripeClient, err := repo.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
			AppID:      appID,
			AppService: repo.appService,
			APIKey:     apiKeySecret.Value,
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
