package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appdb "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// CreateApp creates an app
func (a adapter) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentity.App, error) {
	db := a.client()

	appCreateQuery := db.App.Create().
		SetNamespace(input.Namespace).
		SetName(input.Name).
		SetDescription(input.Description).
		SetType(input.Type).
		SetStatus(appentity.AppStatusReady)

	dbApp, err := appCreateQuery.Save(ctx)
	if err != nil {
		return appentity.AppBase{}, fmt.Errorf("failed to create app: %w", err)
	}

	// Get marketplace listing
	listing, err := a.marketplace.GetListing(ctx, appentity.GetMarketplaceListingInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return appentity.AppBase{}, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	return MapAppBaseFromDB(dbApp, listing), nil
}

// ListApps lists apps
func (a adapter) ListApps(ctx context.Context, params appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error) {
	db := a.client()

	query := db.App.
		Query().
		Where(appdb.Namespace(params.Namespace))

	if params.Type != nil {
		query = query.Where(appdb.Type(*params.Type))
	}

	// Do not return deleted apps by default
	if !params.IncludeDeleted {
		query = query.Where(appdb.DeletedAtIsNil())
	}

	response := pagination.PagedResponse[appentity.App]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]appentity.App, 0, len(paged.Items))
	for _, item := range paged.Items {
		listing, err := a.marketplace.GetListing(ctx, appentity.GetMarketplaceListingInput{
			Type: item.Type,
		})
		if err != nil {
			return response, fmt.Errorf("failed to get listing for app %s: %w", item.ID, err)
		}

		result = append(result, MapAppBaseFromDB(item, listing))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

// GetApp gets an app
func (a adapter) GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error) {
	client := a.client()

	dbApp, err := client.App.Query().
		Where(appdb.Namespace(input.Namespace)).
		Where(appdb.ID(input.ID)).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, app.AppNotFoundError{
				AppID: input,
			}
		}

		return nil, err
	}

	listing, err := a.marketplace.GetListing(ctx, appentity.GetMarketplaceListingInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	return MapAppBaseFromDB(dbApp, listing), nil
}

// GetDefaultApp gets the default app for the app type
func (a adapter) GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error) {
	client := a.client()

	dbApp, err := client.App.Query().
		Where(appdb.Namespace(input.Namespace)).
		Where(appdb.Type(input.Type)).
		Where(appdb.DeletedAtIsNil()).
		First(ctx)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, app.AppDefaultNotFoundError{
				Namespace: input.Namespace,
				Type:      input.Type,
			}
		}

		return nil, err
	}

	listing, err := a.marketplace.GetListing(ctx, appentity.GetMarketplaceListingInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	return MapAppBaseFromDB(dbApp, listing), nil
}

// UninstallApp uninstalls an app
func (a adapter) UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// TODO: Implement uninstall logic
	return fmt.Errorf("uninstall not implemented")
}

func MapAppBaseFromDB(dbApp *db.App, listing appentity.MarketplaceListing) appentity.AppBase {
	return appentity.AppBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:        dbApp.ID,
			Namespace: dbApp.Namespace,
			CreatedAt: dbApp.CreatedAt,
			UpdatedAt: dbApp.UpdatedAt,
			DeletedAt: dbApp.DeletedAt,
		}),
		Type:    dbApp.Type,
		Name:    dbApp.Name,
		Status:  dbApp.Status,
		Listing: listing,
	}
}

// func mapStripeAppFromDB(dbApp *db.App, listing appentity.MarketplaceListing) (app.StripeApp, error) {
// 	appBase := mapAppBaseFromDB(dbApp, listing)

// 	if dbApp.StripeAccountID == nil {
// 		return app.StripeApp{}, fmt.Errorf("stripe account id is nil")
// 	}

// 	if dbApp.StripeLivemode == nil {
// 		return app.StripeApp{}, fmt.Errorf("stripe livemode is nil")
// 	}

// 	return app.StripeApp{
// 		AppBase:         appBase,
// 		StripeAccountId: *dbApp.StripeAccountID,
// 		Livemode:        *dbApp.StripeLivemode,
// 	}, nil
// }
