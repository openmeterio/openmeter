package appstripeadapter

import (
	"context"
	"fmt"

	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

var _ appstripe.AppStripeAdapter = (*adapter)(nil)

// CreateApp creates a new app
func (a adapter) CreateStripeApp(ctx context.Context, input appstripeentity.CreateAppStripeInput) (appstripeentity.StripeApp, error) {
	// Create the app in the database
	appCreateQuery := a.tx.App.Create().
		SetNamespace(input.GetID().Namespace).
		SetName(input.Name).
		SetType(appentity.AppTypeStripe).
		SetStatus(appentity.AppStatusReady)

	dbApp, err := appCreateQuery.Save(ctx)
	if err != nil {
		return appstripeentity.StripeApp{}, fmt.Errorf("failed to create app: %w", err)
	}

	// Create the stripe app in the database
	appStripeCreateQuery := a.tx.AppStripe.Create().
		SetNamespace(input.GetID().Namespace).
		SetApp(dbApp).
		SetStripeAccountID(input.StripeAccountId).
		SetStripeLivemode(input.Livemode)

	dbAppStripe, err := appStripeCreateQuery.Save(ctx)
	if err != nil {
		return appstripeentity.StripeApp{}, fmt.Errorf("failed to create stripe app: %w", err)
	}

	// Set the stripe app edge
	dbAppStripe.Edges.App = dbApp

	return mapAppStripeFromDB(dbAppStripe), nil
}

// mapAppStripeFromDB maps a database stripe app to an app entity
func mapAppStripeFromDB(dbAppStripe *db.AppStripe) appstripeentity.StripeApp {
	return appstripeentity.StripeApp{
		AppBase:         appadapter.MapAppBaseFromDB(dbAppStripe.Edges.App, appstripeentity.StripeMarketplaceListing),
		StripeAccountId: dbAppStripe.StripeAccountID,
		Livemode:        dbAppStripe.StripeLivemode,
	}
}
