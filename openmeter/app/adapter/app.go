package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appdb "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// CreateApp creates an app
func (a adapter) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error) {
	client := a.client()

	appCreateQuery := client.App.Create().
		SetNamespace(input.Namespace).
		SetName(input.Name).
		SetDescription(input.Description).
		SetType(input.Type).
		SetStatus(appentitybase.AppStatusReady)

	dbApp, err := appCreateQuery.Save(ctx)
	if err != nil {
		return appentitybase.AppBase{}, fmt.Errorf("failed to create app: %w", err)
	}

	// Get registry item
	registryItem, err := a.marketplace.Get(ctx, appentity.MarketplaceGetInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return appentitybase.AppBase{}, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	// Map app base from db
	return mapAppBaseFromDB(dbApp, registryItem), nil
}

// ListApps lists apps
func (a adapter) ListApps(ctx context.Context, params appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error) {
	client := a.client()

	query := client.App.
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
	for _, dbApp := range paged.Items {
		registryItem, err := a.marketplace.Get(ctx, appentity.MarketplaceGetInput{
			Type: dbApp.Type,
		})
		if err != nil {
			return response, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
		}

		app, err := mapAppFromDB(ctx, dbApp, registryItem)
		if err != nil {
			return response, fmt.Errorf("failed to map app %s from db: %w", dbApp.ID, err)
		}

		result = append(result, app)
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

	// Get registry item
	registryItem, err := a.marketplace.Get(ctx, appentity.MarketplaceGetInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	// Map app from db
	app, err := mapAppFromDB(ctx, dbApp, registryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to map app from db: %w", err)
	}

	return app, nil
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

	// Get registry item
	registryItem, err := a.marketplace.Get(ctx, appentity.MarketplaceGetInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	// Map app from db
	app, err := mapAppFromDB(ctx, dbApp, registryItem)
	if err != nil {
		return nil, fmt.Errorf("failed to map app from db: %w", err)
	}

	return app, nil
}

// UninstallApp uninstalls an app
func (a adapter) UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// TODO: Implement uninstall logic
	return fmt.Errorf("uninstall not implemented")
}

// mapAppBaseFromDB maps an app base from the database
func mapAppBaseFromDB(dbApp *db.App, registryItem appentity.RegistryItem) appentitybase.AppBase {
	return appentitybase.AppBase{
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
		Listing: registryItem.Listing,
	}
}

// mapAppFromDB maps an app from the database
func mapAppFromDB(ctx context.Context, dbApp *db.App, registryItem appentity.RegistryItem) (appentity.App, error) {
	appBase := mapAppBaseFromDB(dbApp, registryItem)

	app, err := registryItem.Factory.NewApp(ctx, appBase)
	if err != nil {
		return app, fmt.Errorf("failed to create app with %s factory: %w", appBase.Type, err)
	}

	return app, nil
}
