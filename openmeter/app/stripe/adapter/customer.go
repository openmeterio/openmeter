package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// GetStripeCustomerData gets stripe customer data
func (a *adapter) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.CustomerData{}, models.NewGenericValidationError(
			fmt.Errorf("error getting stripe customer data: %w", err),
		)
	}

	stripeCustomerDBEntity, err := a.db.AppStripeCustomer.
		Query().
		Where(appstripecustomerdb.Namespace(input.AppID.Namespace)).
		Where(appstripecustomerdb.AppID(input.AppID.ID)).
		Where(appstripecustomerdb.CustomerID(input.CustomerID.ID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.CustomerData{}, app.NewAppCustomerPreConditionError(
				input.AppID,
				app.AppTypeStripe,
				&input.CustomerID,
				"customer has no data for stripe app",
			)
		}

		return appstripeentity.CustomerData{}, fmt.Errorf("error getting stripe customer data: %w", err)
	}

	customerData := appstripeentity.CustomerData{
		StripeCustomerID:             stripeCustomerDBEntity.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerDBEntity.StripeDefaultPaymentMethodID,
	}

	if err := customerData.Validate(); err != nil {
		return appstripeentity.CustomerData{}, fmt.Errorf("error validating stripe customer data: %w", err)
	}

	return customerData, nil
}

// UpsertStripeCustomerData upserts stripe customer data
func (a *adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			fmt.Errorf("error upsert stripe customer data: %w", err),
		)
	}

	// Get the stripe app client
	stripeAppData, stripeAppClient, err := a.getStripeAppClient(ctx, input.AppID, "upsertStripeCustomerData", "customer_id", input.CustomerID.ID, "stripe_customer_id", input.StripeCustomerID)
	if err != nil {
		return fmt.Errorf("failed to get stripe app client: %w", err)
	}

	// Check if the Stripe customer exists in the stripe account
	_, err = stripeAppClient.GetCustomer(ctx, input.StripeCustomerID)
	if err != nil {
		if _, ok := err.(stripeclient.StripeCustomerNotFoundError); ok {
			return app.NewAppCustomerPreConditionError(
				input.AppID,
				app.AppTypeStripe,
				&input.CustomerID,
				fmt.Sprintf("stripe customer %s not found in stripe account: %s", input.StripeCustomerID, stripeAppData.StripeAccountID),
			)
		}

		return fmt.Errorf("failed to get stripe customer: %w", err)
	}

	// Check if the Stripe payment method exists in the stripe account
	if input.StripeDefaultPaymentMethodID != nil {
		paymentMethod, err := stripeAppClient.GetPaymentMethod(ctx, *input.StripeDefaultPaymentMethodID)
		if err != nil {
			if _, ok := err.(stripeclient.StripePaymentMethodNotFoundError); ok {
				return app.NewAppProviderPreConditionError(
					input.AppID,
					fmt.Sprintf("stripe payment method %s not found in stripe account: %s", *input.StripeDefaultPaymentMethodID, stripeAppData.StripeAccountID),
				)
			}

			return fmt.Errorf("failed to get stripe payment method: %w", err)
		}

		// Check if the payment method belongs to the customer
		if paymentMethod.StripeCustomerID == nil || *paymentMethod.StripeCustomerID != input.StripeCustomerID {
			return app.NewAppProviderPreConditionError(
				input.AppID,
				fmt.Sprintf(
					"stripe payment method %s does not belong to stripe customer %s in stripe account: %s",
					*input.StripeDefaultPaymentMethodID,
					input.StripeCustomerID,
					stripeAppData.StripeAccountID,
				),
			)
		}
	}

	// Start transaction
	_, err = entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (any, error) {
		// Make sure the customer has an app relationship
		err := a.appService.EnsureCustomer(ctx, app.EnsureCustomerInput{
			AppID:      input.AppID,
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to ensure customer: %w", err)
		}

		// Upsert stripe customer data
		err = repo.db.AppStripeCustomer.
			Create().
			SetNamespace(input.AppID.Namespace).
			SetStripeAppID(input.AppID.ID).
			SetCustomerID(input.CustomerID.ID).
			SetStripeCustomerID(input.StripeCustomerID).
			SetNillableStripeDefaultPaymentMethodID(input.StripeDefaultPaymentMethodID).
			// Upsert
			OnConflictColumns(appstripecustomerdb.FieldNamespace, appstripecustomerdb.FieldAppID, appstripecustomerdb.FieldCustomerID).
			UpdateStripeCustomerID().
			Exec(ctx)
		if err != nil {
			if entdb.IsConstraintError(err) {
				return nil, app.NewAppCustomerPreConditionError(
					input.AppID,
					app.AppTypeStripe,
					&input.CustomerID,
					"unique stripe customer id",
				)
			}

			return nil, fmt.Errorf("failed to upsert app stripe customer data: %w", err)
		}

		return nil, nil
	})

	return err
}

// DeleteStripeCustomerData deletes stripe customer data
func (a *adapter) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(
			fmt.Errorf("error delete stripe customer data: %w", err),
		)
	}

	// Determine namespace
	var namespace string

	if input.AppID != nil {
		namespace = input.AppID.Namespace
	}

	if input.CustomerID != nil {
		namespace = input.CustomerID.Namespace
	}

	if namespace == "" {
		return models.NewGenericValidationError(
			fmt.Errorf("error delete stripe customer data: namespace is empty"),
		)
	}

	// Start transaction
	_, err := entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (any, error) {
		// Delete stripe app customer data
		query := repo.db.AppStripeCustomer.
			Delete().
			Where(
				appstripecustomerdb.Namespace(namespace),
			)

		if input.CustomerID != nil {
			query = query.Where(appstripecustomerdb.CustomerID(input.CustomerID.ID))
		}

		if input.AppID != nil {
			query = query.Where(appstripecustomerdb.AppID(input.AppID.ID))
		}

		_, err := query.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete app stripe customer data: %w", err)
		}

		// Delete app customer relationship
		err = a.appService.DeleteCustomer(ctx, app.DeleteCustomerInput{
			AppID:      input.AppID,
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to delete customer relationship: %w", err)
		}

		return nil, nil
	})
	return err
}

// createStripeCustomer creates a new stripe customer
func (a *adapter) createStripeCustomer(ctx context.Context, input appstripeentity.CreateStripeCustomerInput) (appstripeentity.CreateStripeCustomerOutput, error) {
	// Get the stripe app
	stripeAppData, err := a.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: input.AppID,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.secretService.GetAppSecret(ctx, stripeAppData.APIKey)
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.stripeAppClientFactory(stripeclient.StripeAppClientConfig{
		AppID:      input.AppID,
		AppService: a.appService,
		APIKey:     apiKeySecret.Value,
		Logger:     a.logger.With("operation", "createStripeCustomer", "app_id", input.AppID.ID, "customer_id", input.CustomerID.ID),
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Create stripe customer
	stripeCustomer, err := stripeClient.CreateCustomer(ctx, stripeclient.CreateStripeCustomerInput{
		AppID:      input.AppID,
		CustomerID: input.CustomerID,
		Name:       input.Name,
		Email:      input.Email,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to create stripe customer: %w", err)
	}

	// Upsert stripe customer data
	err = a.UpsertStripeCustomerData(ctx, appstripeentity.UpsertStripeCustomerDataInput{
		AppID:            input.AppID,
		CustomerID:       input.CustomerID,
		StripeCustomerID: stripeCustomer.StripeCustomerID,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to upsert stripe customer data: %w", err)
	}

	// Output
	out := appstripeentity.CreateStripeCustomerOutput{
		StripeCustomerID: stripeCustomer.StripeCustomerID,
	}

	if err := out.Validate(); err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to validate create stripe customer output: %w", err)
	}

	return out, nil
}
