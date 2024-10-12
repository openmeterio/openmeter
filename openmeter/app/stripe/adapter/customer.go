package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripedb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripe"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// GetStripeCustomerData gets stripe customer data
func (a adapter) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerAppData, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.CustomerAppData{}, appstripe.ValidationError{
			Err: fmt.Errorf("error getting stripe customer data: %w", err),
		}
	}

	stripeCustomerDBEntity, err := a.db.AppStripeCustomer.
		Query().
		Where(appstripecustomerdb.Namespace(input.AppID.Namespace)).
		Where(appstripecustomerdb.AppID(input.AppID.ID)).
		Where(appstripecustomerdb.CustomerID(input.CustomerID.ID)).
		Only(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return appstripeentity.CustomerAppData{}, app.CustomerPreConditionError{
				AppID:      input.AppID,
				AppType:    appentitybase.AppTypeStripe,
				CustomerID: input.CustomerID,
				Condition:  "customer has no data for stripe app",
			}
		}

		return appstripeentity.CustomerAppData{}, fmt.Errorf("error getting stripe customer data: %w", err)
	}

	return appstripeentity.CustomerAppData{
		StripeCustomerID:             stripeCustomerDBEntity.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerDBEntity.StripeDefaultPaymentMethodID,
	}, nil
}

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

// createStripeCustomer creates a new stripe customer
func (a adapter) createStripeCustomer(ctx context.Context, input appstripeentity.CreateStripeCustomerInput) (appstripeentity.CreateStripeCustomerOutput, error) {
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
			ID:        stripeApp.APIKey,
		},
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.stripeClientFactory(stripeclient.StripeClientConfig{
		Namespace: stripeApp.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return appstripeentity.CreateStripeCustomerOutput{}, fmt.Errorf("failed to create stripe client: %w", err)
	}

	// Create stripe customer
	stripeCustomer, err := stripeClient.CreateCustomer(ctx, stripeclient.CreateStripeCustomerInput{
		AppID:      input.AppID,
		CustomerID: input.CustomerID,
		Name:       input.Name,
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
