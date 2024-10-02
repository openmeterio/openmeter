package appstripeadapter

import (
	"context"
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
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

		return mapAppStripeFromDB(a.db, appBase, dbAppStripe), nil
	})
}

// UpsertStripeCustomerData upserts stripe customer data
func (a adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
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

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(client *entdb.Client, appBase appentitybase.AppBase, dbAppStripe *db.AppStripe) appstripeentity.App {
	return appstripeentity.App{
		AppBase: appBase,
		Client:  client,

		StripeAccountId: dbAppStripe.StripeAccountID,
		Livemode:        dbAppStripe.StripeLivemode,
	}
}
