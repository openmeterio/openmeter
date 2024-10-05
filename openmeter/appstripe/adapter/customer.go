package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// UpsertStripeCustomerData upserts stripe customer data
func (a adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error upsert stripe customer data: %w", err),
		}
	}

	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		err := a.customerService.UpsertAppCustomer(ctx, customerentity.UpsertAppCustomerInput{
			AppID:      input.AppID,
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert app customer: %w", err)
		}

		err = a.db.AppStripeCustomer.
			Create().
			SetNamespace(input.AppID.Namespace).
			SetStripeAppID(input.AppID.ID).
			SetCustomerID(input.CustomerID.ID).
			SetStripeCustomerID(input.StripeCustomerID).
			// Upsert
			OnConflictColumns(appstripecustomerdb.FieldNamespace, appstripecustomerdb.FieldAppID, appstripecustomerdb.FieldCustomerID).
			UpdateStripeCustomerID().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to upsert app stripe customer data: %w", err)
		}

		return nil
	})
}

// DeleteStripeCustomerData deletes stripe customer data
func (a adapter) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error delete stripe customer data: %w", err),
		}
	}

	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		query := a.db.AppStripeCustomer.
			Delete().
			Where(
				appstripecustomerdb.Namespace(input.CustomerID.Namespace),
				appstripecustomerdb.CustomerID(input.CustomerID.ID),
			)

		if input.AppID != nil {
			query = query.Where(appstripecustomerdb.AppID(input.AppID.ID))
		}

		_, err := query.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete app stripe customer data: %w", err)
		}

		return nil
	})
}

// CreateStripeCustomer creates a new stripe customer
func (a adapter) CreateStripeCustomer(ctx context.Context, input appstripeentity.CreateStripeCustomerInput) (appstripeentity.CreateStripeCustomerOutput, error) {
	// Get the stripe app
	stripeApp, err := a.db.AppStripe.
		Query().
		Where(appstripedb.ID(input.AppID.ID)).
		Where(appstripedb.Namespace(input.AppID.Namespace)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.CreateStripeCustomerOutput{}, app.AppNotFoundError{
				AppID: input.AppID,
			}
		}

		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to get stripe app: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.secretService.GetAppSecret(ctx, secretentity.GetAppSecretInput{
		NamespacedID: models.NamespacedID{
			Namespace: stripeApp.Namespace,
			ID:        stripeApp.ID,
		},
		Key: *stripeApp.APIKey,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.stripeClientFactory(appstripeentity.StripeClientConfig{
		Namespace: stripeApp.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Create stripe customer
	stripeCustomer, err := stripeClient.CreateCustomer(ctx, input)
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
