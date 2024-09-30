package appstripeadapter

import (
	"context"
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.App, error) {
	client := a.client()

	app, err := a.appService.CreateApp(ctx, appentity.CreateAppInput{
		Namespace:   input.Namespace,
		Name:        input.Name,
		Description: input.Description,
		Type:        appentitybase.AppTypeStripe,
	})
	if err != nil {
		return appstripeentity.App{}, fmt.Errorf("failed to create app: %w", err)
	}

	// Create the stripe app in the database
	appStripeCreateQuery := client.AppStripe.Create().
		SetID(app.GetID().ID).
		SetNamespace(input.Namespace).
		SetStripeAccountID(input.StripeAccountID).
		SetStripeLivemode(input.Livemode)

	dbAppStripe, err := appStripeCreateQuery.Save(ctx)
	if err != nil {
		return appstripeentity.App{}, fmt.Errorf("failed to create stripe app: %w", err)
	}

	return mapAppStripeFromDB(app, dbAppStripe), nil
}

// UpsertStripeCustomerData upserts stripe customer data
func (a adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	client := a.client()

	err := a.appCustomerService.UpsertAppCustomer(ctx, appcustomerentity.UpsertAppCustomerInput{
		AppID:      input.AppID,
		CustomerID: input.CustomerID.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert app customer: %w", err)
	}

	err = client.AppStripeCustomer.
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
}

// DeleteStripeCustomerData deletes stripe customer data
func (a adapter) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	client := a.client()

	query := client.AppStripeCustomer.
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
}

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(app appentity.App, dbAppStripe *db.AppStripe) appstripeentity.App {
	return appstripeentity.App{
		AppBase: app.GetAppBase(),
		// TODO: add Client

		StripeAccountId: dbAppStripe.StripeAccountID,
		Livemode:        dbAppStripe.StripeLivemode,
	}
}
