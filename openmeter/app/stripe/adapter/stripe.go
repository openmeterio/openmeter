package appstripeadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
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
func (a *adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.AppBase, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.AppBase{}, models.NewGenericValidationError(
			fmt.Errorf("error create stripe app: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.AppBase, error) {
		// Create the base app
		appBase, err := repo.appService.CreateApp(ctx, input.CreateAppInput)
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
			SetWebhookSecret(input.WebhookSecret.ID).
			SetMaskedAPIKey(input.MaskedAPIKey)

		dbApp, err := appStripeCreateQuery.Save(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return appstripeentity.AppBase{}, models.NewGenericConflictError(
					fmt.Errorf("stripe app already exists with stripe account id: %s in namespace %s", input.StripeAccountID, appBase.GetID().Namespace),
				)
			}

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
func (a *adapter) UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyAdapterInput) error {
	// Validate the input
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			fmt.Errorf("error replace api key: %w", err),
		)
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
		Logger:     a.logger.With("operation", "validateStripeAPIKey", "app_id", input.AppID.ID),
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

		return models.NewGenericValidationError(
			err,
		)
	}

	// Check if it belongs to the same stripe account
	stripeAccount, err := stripeClient.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate stripe api key: %w", err)
	}

	// Check if the stripe account id matches with the stored one
	if stripeAccount.StripeAccountID != appData.StripeAccountID {
		return models.NewGenericValidationError(
			fmt.Errorf("stripe account id mismatch: %s != %s", stripeAccount.StripeAccountID, appData.StripeAccountID),
		)
	}

	// Update the API key
	newApiKeySecretID, err := a.secretService.UpdateAppSecret(ctx, secretentity.UpdateAppSecretInput{
		AppID:    input.AppID,
		SecretID: appData.APIKey,
		Key:      appstripeentity.APIKeySecretKey,
		Value:    input.APIKey,
	})
	if err != nil {
		return fmt.Errorf("failed to update api key app secret: %w", err)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		// Update the API key in the database if it has changed
		// Some secrets stores don't update the id when updating the value
		if appData.APIKey.ID != newApiKeySecretID.ID {
			err = repo.db.AppStripe.Update().
				Where(appstripedb.Namespace(input.AppID.Namespace)).
				Where(appstripedb.ID(input.AppID.ID)).
				SetAPIKey(newApiKeySecretID.ID).
				SetMaskedAPIKey(input.MaskedAPIKey).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to update api key: %w", err)
			}
		}

		// Update the app status to ready
		status := app.AppStatusReady

		err = a.appService.UpdateAppStatus(ctx, app.UpdateAppStatusInput{
			ID:     input.AppID,
			Status: status,
		})
		if err != nil {
			return fmt.Errorf("failed to update app status to %s for %s: %w", input.AppID.ID, status, err)
		}

		return nil
	})
}

// GetStripeAppData gets stripe customer data
func (a *adapter) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.AppData{}, models.NewGenericValidationError(
			fmt.Errorf("error getting stripe customer data: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.AppData, error) {
		dbApp, err := repo.db.AppStripe.
			Query().
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Where(appstripedb.ID(input.AppID.ID)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return appstripeentity.AppData{}, app.NewAppNotFoundError(input.AppID)
			}

			return appstripeentity.AppData{}, fmt.Errorf("error getting stripe customer data: %w", err)
		}

		// Map the database stripe app to an app entity
		appData := mapAppStripeData(input.AppID, dbApp)
		if err := appData.Validate(); err != nil {
			return appstripeentity.AppData{}, models.NewGenericValidationError(fmt.Errorf("error validating stripe app data: %w", err))
		}

		return appData, nil
	})
}

// DeleteStripeAppData deletes the stripe app data
func (a *adapter) DeleteStripeAppData(ctx context.Context, input appstripeentity.DeleteStripeAppDataInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			fmt.Errorf("error delete stripe app: %w", err),
		)
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, repo *adapter) error {
		// Delete the stripe app data
		_, err := repo.db.AppStripe.
			Delete().
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Where(appstripedb.ID(input.AppID.ID)).
			Exec(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return app.NewAppNotFoundError(input.AppID)
			}

			return fmt.Errorf("failed to delete stripe app: %w", err)
		}

		return nil
	})
}

// GetWebhookSecret gets the webhook secret
func (a *adapter) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	if err := input.Validate(); err != nil {
		return secretentity.Secret{}, models.NewGenericValidationError(
			fmt.Errorf("error get webhook secret: %w", err),
		)
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
				// We don't know the namespace from the app id for webhook requests
				return secretentity.Secret{}, app.NewAppNotFoundError(app.AppID{
					Namespace: "",
					ID:        input.AppID,
				})
			}

			return secretentity.Secret{}, fmt.Errorf("failed to get stripe app: %w", err)
		}

		// Get the webhook secret
		appID := app.AppID{
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
func (a *adapter) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, models.NewGenericValidationError(
			fmt.Errorf("error set customer default payment method: %w", err),
		)
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
				return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, app.NewAppCustomerPreConditionError(
					input.AppID,
					app.AppTypeStripe,
					nil,
					fmt.Sprintf("stripe customer has no data for stripe app: %s", input.StripeCustomerID),
				)
			}
		}

		customerID := customer.CustomerID{
			Namespace: input.AppID.Namespace,
			ID:        appCustomer.CustomerID,
		}

		// Check if the stripe customer id matches with the input
		if appCustomer.StripeCustomerID != input.StripeCustomerID {
			return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, app.NewAppCustomerPreConditionError(
				input.AppID,
				app.AppTypeStripe,
				&customerID,
				"customer stripe customer id mismatch",
			)
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
func (a *adapter) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.CreateCheckoutSessionOutput{}, models.NewGenericValidationError(
			fmt.Errorf("error create checkout session: %w", err),
		)
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (appstripeentity.CreateCheckoutSessionOutput, error) {
		// Get the stripe app
		stripeApp, err := repo.db.AppStripe.
			Query().
			Where(appstripedb.ID(input.AppID.ID)).
			Where(appstripedb.Namespace(input.AppID.Namespace)).
			Only(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return appstripeentity.CreateCheckoutSessionOutput{}, app.NewAppNotFoundError(input.AppID)
			}

			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe app: %w", err)
		}

		// Get or create customer
		var targetCustomer *customer.Customer

		if input.CustomerID != nil {
			targetCustomer, err = repo.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: input.CustomerID,
			})
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get customer: %w", err)
			}
		}

		// Create a customer if create input is provided
		if input.CreateCustomerInput != nil {
			targetCustomer, err = repo.customerService.CreateCustomer(ctx, *input.CreateCustomerInput)
			if err != nil {
				return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create customer: %w", err)
			}
		}

		customerID := targetCustomer.GetID()

		// Get the stripe app customer
		var stripeCustomerId string
		{
			stripeAppCustomer, err := repo.db.AppStripeCustomer.
				Query().
				Where(appstripecustomerdb.AppID(input.AppID.ID)).
				Where(appstripecustomerdb.Namespace(input.AppID.Namespace)).
				Where(appstripecustomerdb.CustomerID(customerID.ID)).
				Only(ctx)
			if err != nil {
				if entdb.IsNotFound(err) {
					// If Stripe Customer ID is provided we need to upsert it
					if input.StripeCustomerID != nil {
						err = a.UpsertStripeCustomerData(ctx, appstripeentity.UpsertStripeCustomerDataInput{
							AppID:            input.AppID,
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
							AppID:      input.AppID,
							CustomerID: customerID,
							Name:       &targetCustomer.Name,
							Email:      targetCustomer.PrimaryEmail,
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
		apiKeySecret, err := repo.secretService.GetAppSecret(ctx, secretentity.NewSecretID(input.AppID, stripeApp.APIKey, appstripeentity.APIKeySecretKey))
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
		}

		// Stripe Client
		stripeClient, err := repo.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
			AppID:      input.AppID,
			AppService: repo.appService,
			APIKey:     apiKeySecret.Value,
			Logger:     a.logger.With("operation", "createCheckoutSession", "app_id", input.AppID.ID, "customer_id", customerID.ID),
		})
		if err != nil {
			return appstripeentity.CreateCheckoutSessionOutput{}, fmt.Errorf("failed to create stripe client: %w", err)
		}

		// Set the currency if customer has one and it is not provided
		if input.Options.Currency == nil && targetCustomer.Currency != nil {
			input.Options.Currency = stripeclient.CurrencyPtr(targetCustomer.Currency)
		}

		// Create the checkout session
		checkoutSession, err := stripeClient.CreateCheckoutSession(ctx, stripeclient.CreateCheckoutSessionInput{
			StripeCustomerID: stripeCustomerId,
			AppID:            input.AppID,
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
			AppID:                 input.AppID,
			CustomerID:            customerID,
			StripeCustomerID:      stripeCustomerId,
			StripeCheckoutSession: checkoutSession,
		}, nil
	})
}

// GetSupplierContact returns a supplier contact for the app
func (a adapter) GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return billing.SupplierContact{}, models.NewGenericValidationError(
			fmt.Errorf("error validate input: %w", err),
		)
	}

	// Get stripe app data
	stripeAppData, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput(input))
	if err != nil {
		return billing.SupplierContact{}, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Test mode Stripe accounts do not have supplier contact information
	if !stripeAppData.Livemode {
		return billing.SupplierContact{
			// TODO: use organization name
			Name: "Stripe Test Account",
			Address: models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
		}, nil
	}

	// Get Stripe App client
	_, stripeAppClient, err := a.getStripeAppClient(ctx, input.AppID, "getSupplierContact", "app_id", input.AppID.ID)
	if err != nil {
		return billing.SupplierContact{}, fmt.Errorf("failed to get stripe app client: %w", err)
	}

	// Get Stripe Account
	stripeAccount, err := stripeAppClient.GetAccount(ctx)
	if err != nil {
		return billing.SupplierContact{}, fmt.Errorf("failed to get stripe account: %w", err)
	}

	if stripeAccount.BusinessProfile == nil || stripeAccount.BusinessProfile.Name == "" {
		return billing.SupplierContact{}, app.NewAppProviderPreConditionError(
			input.AppID,
			fmt.Sprintf("stripe account is missing business profile name: %s", stripeAccount.StripeAccountID),
		)
	}

	if stripeAccount.Country == "" {
		return billing.SupplierContact{}, app.NewAppProviderPreConditionError(
			input.AppID,
			fmt.Sprintf("stripe account country is empty: %s", stripeAccount.StripeAccountID),
		)
	}

	supplierContact := billing.SupplierContact{
		Name: stripeAccount.BusinessProfile.Name,
		Address: models.Address{
			Country: &stripeAccount.Country,
		},
	}

	return supplierContact, nil
}

func (a adapter) GetStripeInvoice(ctx context.Context, input appstripeentity.GetStripeInvoiceInput) (*stripe.Invoice, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("error validate input: %w", err),
		)
	}

	// Get Stripe App client
	_, stripeAppClient, err := a.getStripeAppClient(ctx, input.AppID, "getStripeInvoice", "app_id", input.AppID.ID, "stripe_invoice_id", input.StripeInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe app client: %w", err)
	}

	// Get the invoice
	return stripeAppClient.GetInvoice(ctx, stripeclient.GetInvoiceInput{
		StripeInvoiceID: input.StripeInvoiceID,
	})
}

// CreatePortalSession creates a portal session for a customer.
func (a adapter) CreatePortalSession(ctx context.Context, input appstripeentity.CreateStripePortalSessionInput) (appstripeentity.StripePortalSession, error) {
	// Validate input
	if err := input.Validate(); err != nil {
		return appstripeentity.StripePortalSession{}, models.NewGenericValidationError(
			fmt.Errorf("error validate input: %w", err),
		)
	}

	// Get Stripe App client
	_, stripeAppClient, err := a.getStripeAppClient(ctx, input.AppID, "createPortalSession", "app_id", input.AppID.ID)
	if err != nil {
		return appstripeentity.StripePortalSession{}, fmt.Errorf("failed to get stripe app client: %w", err)
	}

	// Get the stripe app data
	stripeCustomerData, err := a.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      input.AppID,
		CustomerID: input.CustomerID,
	})
	if err != nil {
		return appstripeentity.StripePortalSession{}, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	if stripeCustomerData.StripeCustomerID == "" {
		return appstripeentity.StripePortalSession{}, app.NewAppCustomerPreConditionError(
			input.AppID,
			app.AppTypeStripe,
			&input.CustomerID,
			"stripe customer id is empty",
		)
	}

	// Create the portal session
	portalSession, err := stripeAppClient.CreatePortalSession(ctx, stripeclient.CreatePortalSessionInput{
		StripeCustomerID: stripeCustomerData.StripeCustomerID,
		ConfigurationID:  input.ConfigurationID,
		ReturnURL:        input.ReturnURL,
	})
	if err != nil {
		return appstripeentity.StripePortalSession{}, fmt.Errorf("failed to create portal session: %w", err)
	}

	return appstripeentity.StripePortalSession{
		ID:               portalSession.ID,
		Configuration:    portalSession.Configuration,
		StripeCustomerID: stripeCustomerData.StripeCustomerID,
		Livemode:         portalSession.Livemode,
		Locale:           portalSession.Locale,
		ReturnURL:        portalSession.ReturnURL,
		URL:              portalSession.URL,
		CreatedAt:        portalSession.CreatedAt,
	}, nil
}

// getStripeAppClient returns a Stripe App Client based on App ID
func (a adapter) getStripeAppClient(ctx context.Context, appID app.AppID, logOperation string, logFields ...any) (appstripeentity.AppData, stripeclient.StripeAppClient, error) {
	// Validate app id
	if err := appID.Validate(); err != nil {
		return appstripeentity.AppData{}, nil, fmt.Errorf("app id: %w", err)
	}

	// Get the stripe app data
	stripeAppData, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: appID,
	})
	if err != nil {
		return stripeAppData, nil, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.secretService.GetAppSecret(ctx, stripeAppData.APIKey)
	if err != nil {
		return stripeAppData, nil, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
		AppID:      appID,
		AppService: a.appService,
		APIKey:     apiKeySecret.Value,
		Logger:     a.logger.With("operation", logOperation).With(logFields...),
	})
	if err != nil {
		return stripeAppData, nil, fmt.Errorf("failed to create stripe client: %w", err)
	}

	return stripeAppData, stripeClient, nil
}

// mapAppStripeData maps stripe app data from the database
func mapAppStripeData(appID app.AppID, dbApp *entdb.AppStripe) appstripeentity.AppData {
	return appstripeentity.AppData{
		StripeAccountID: dbApp.StripeAccountID,
		Livemode:        dbApp.StripeLivemode,
		APIKey:          secretentity.NewSecretID(appID, dbApp.APIKey, appstripeentity.APIKeySecretKey),
		MaskedAPIKey:    dbApp.MaskedAPIKey,
		StripeWebhookID: dbApp.StripeWebhookID,
		WebhookSecret:   secretentity.NewSecretID(appID, dbApp.WebhookSecret, appstripeentity.WebhookSecretKey),
	}
}
