package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appdb "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// CreateApp creates an app
func (a adapter) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (appentitybase.AppBase, error) {
			count, err := repo.db.App.Query().
				Where(appdb.Namespace(input.Namespace)).
				Where(appdb.Type(input.Type)).
				Count(ctx)
			if err != nil {
				return appentitybase.AppBase{}, fmt.Errorf("failed to count apps from same type: %w", err)
			}

			appCreateQuery := repo.db.App.Create().
				SetNamespace(input.Namespace).
				SetName(input.Name).
				SetDescription(input.Description).
				SetType(input.Type).
				// Set the app as default if it is the first app of its type
				SetIsDefault(count == 0).
				SetStatus(appentitybase.AppStatusReady)

			dbApp, err := appCreateQuery.Save(ctx)
			if err != nil {
				return appentitybase.AppBase{}, fmt.Errorf("failed to create app: %w", err)
			}

			// Get registry item
			registryItem, err := repo.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
				Type: dbApp.Type,
			})
			if err != nil {
				return appentitybase.AppBase{}, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
			}

			// Map app base from db
			return mapAppBaseFromDB(dbApp, registryItem), nil
		})
}

// UpdateAppStatus updates an app status
func (a adapter) UpdateAppStatus(ctx context.Context, input appentity.UpdateAppStatusInput) error {
	_, err := a.db.App.Update().
		Where(appdb.Namespace(input.ID.Namespace)).
		Where(appdb.ID(input.ID.ID)).
		SetStatus(input.Status).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update app status: %w", err)
	}

	return nil
}

// ListApps lists apps
func (a adapter) ListApps(ctx context.Context, params appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (pagination.PagedResponse[appentity.App], error) {
			query := repo.db.App.
				Query().
				Where(appdb.Namespace(params.Namespace))

			if params.Type != nil {
				query = query.Where(appdb.Type(*params.Type))
			}

			// Do not return deleted apps by default
			if !params.IncludeDeleted {
				query = query.Where(appdb.DeletedAtIsNil())
			}

			// Only list apps that has customer data for the given customer
			if params.CustomerID != nil {
				query = query.Where(appdb.HasCustomerAppsWith(appcustomerdb.CustomerID(params.CustomerID.ID)))
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
				registryItem, err := repo.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
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
		},
	)
}

// GetApp gets an app
func (a adapter) GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (appentity.App, error) {
			dbApp, err := repo.db.App.Query().
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
			registryItem, err := repo.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
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
		},
	)
}

// GetDefaultApp gets the default app for the app type
func (a adapter) GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (appentity.App, error) {
			dbApp, err := repo.db.App.Query().
				Where(appdb.Namespace(input.Namespace)).
				Where(appdb.Type(input.Type)).
				Where(appdb.IsDefault(true)).
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
			registryItem, err := repo.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
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
		})
}

// UninstallApp uninstalls an app
func (a adapter) UninstallApp(ctx context.Context, input appentity.UninstallAppInput) error {
	_, err := entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (any, error) {
		app, err := repo.GetApp(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get app: %w", err)
		}

		// Get app factory through registry
		registryItem, err := repo.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
			Type: app.GetType(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get listing for app: %w", err)
		}

		// Uninstall app through factory
		err = registryItem.Factory.UninstallApp(ctx, app.GetID())
		if err != nil {
			return nil, fmt.Errorf("failed to uninstall app: %w", err)
		}

		// Delete app, app types and customer app data is a cascading delete
		// we don't need to clean them up specifically
		deleteCount, err := repo.db.App.Delete().
			Where(appdb.Namespace(input.Namespace)).
			Where(appdb.ID(input.ID)).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete app from database: %w", err)
		}

		// Should be exactly one
		if deleteCount != 1 {
			return nil, fmt.Errorf(
				"inconsistent app delete for %s in namespace %s, count: %d",
				app.GetID().ID,
				app.GetID().Namespace,
				deleteCount,
			)
		}

		return nil, nil
	})
	return err
}

// mapAppBaseFromDB maps an app base from the database
func mapAppBaseFromDB(dbApp *db.App, registryItem appentity.RegistryItem) appentitybase.AppBase {
	return appentitybase.AppBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
			ID:          dbApp.ID,
			Namespace:   dbApp.Namespace,
			CreatedAt:   dbApp.CreatedAt,
			UpdatedAt:   dbApp.UpdatedAt,
			DeletedAt:   dbApp.DeletedAt,
			Name:        dbApp.Name,
			Description: dbApp.Description,
		}),
		Type:     dbApp.Type,
		Status:   dbApp.Status,
		Listing:  registryItem.Listing,
		Metadata: dbApp.Metadata,
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
