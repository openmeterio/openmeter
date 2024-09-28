package appstripeadapter

import (
	"context"
	"fmt"

	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	// Create the app in the database
	appCreateQuery := a.tx.App.Create().
		SetNamespace(input.GetID().Namespace).
		SetName(input.Name).
		SetType(appentity.AppTypeStripe).
		SetStatus(appentity.AppStatusReady)

	dbApp, err := appCreateQuery.Save(ctx)
	if err != nil {
		return appstripeentity.App{}, fmt.Errorf("failed to create app: %w", err)
	}

	// Create the stripe app in the database
	appStripeCreateQuery := a.tx.AppStripe.Create().
		SetNamespace(input.GetID().Namespace).
		SetApp(dbApp).
		SetStripeAccountID(input.StripeAccountId).
		SetStripeLivemode(input.Livemode)

	dbAppStripe, err := appStripeCreateQuery.Save(ctx)
	if err != nil {
		return appstripeentity.App{}, fmt.Errorf("failed to create stripe app: %w", err)
	}

	// Set the stripe app edge
	dbAppStripe.Edges.App = dbApp

	return mapAppStripeFromDB(dbAppStripe), nil
}

// UpsertStripeCustomerData upserts stripe customer data
func (a adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	err := a.tx.AppStripeCustomer.
		Create().
		SetNamespace(input.CustomerID.Namespace).
		SetAppID(input.AppID.ID).
		SetCustomerID(input.StripeCustomerID).
		SetStripeCustomerID(input.StripeCustomerID).
		// Upsert
		OnConflict().
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert app stripe customer data: %w", err)
	}

	return nil
}

// DeleteStripeCustomerData deletes stripe customer data
func (a adapter) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	_, err := a.tx.AppStripeCustomer.
		Delete().
		Where(
			appstripecustomerdb.Namespace(input.CustomerID.Namespace),
			appstripecustomerdb.AppID(input.AppID.ID),
			appstripecustomerdb.CustomerID(input.CustomerID.ID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete app stripe customer data: %w", err)
	}

	return nil
}

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(dbAppStripe *db.AppStripe) appstripeentity.App {
	return appstripeentity.App{
		AppBase:         appadapter.MapAppBaseFromDB(dbAppStripe.Edges.App, appstripeentity.StripeMarketplaceListing),
		StripeAccountId: dbAppStripe.StripeAccountID,
		Livemode:        dbAppStripe.StripeLivemode,
	}
}
