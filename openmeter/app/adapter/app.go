package appadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appdb "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// CreateApp creates an app
func (a *adapter) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.AppBase, error) {
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *adapter) (app.AppBase, error) {
				appCreateQuery := repo.db.App.Create().
					SetNamespace(input.Namespace).
					SetName(input.Name).
					SetDescription(input.Description).
					SetType(input.Type).
					SetStatus(app.AppStatusReady)

				// Set ID if provided by the input
				if input.ID != nil {
					appCreateQuery = appCreateQuery.SetID(input.ID.ID)
				}

				dbApp, err := appCreateQuery.Save(ctx)
				if err != nil {
					return app.AppBase{}, fmt.Errorf("failed to create app: %w", err)
				}

				// Get registry item
				registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
					Type: dbApp.Type,
				})
				if err != nil {
					return app.AppBase{}, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
				}

				// Map app base from db
				return mapAppBaseFromDB(dbApp, registryItem), nil
			})
	})
}

// UpdateAppStatus updates an app status
func (a *adapter) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
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
func (a *adapter) ListApps(ctx context.Context, params app.ListAppInput) (pagination.Result[app.App], error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (pagination.Result[app.App], error) {
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
				query = query.Where(appdb.HasCustomerAppsWith(
					appcustomerdb.CustomerID(params.CustomerID.ID),
					appcustomerdb.DeletedAtIsNil(),
				))
			}

			// Only list apps that has the given app IDs
			if len(params.AppIDs) > 0 {
				appIDs := lo.Map(params.AppIDs, func(appID app.AppID, _ int) string {
					return appID.ID
				})

				query = query.Where(appdb.IDIn(appIDs...))
			}

			response := pagination.Result[app.App]{
				Page: params.Page,
			}

			paged, err := query.Paginate(ctx, params.Page)
			if err != nil {
				return response, err
			}

			result := make([]app.App, 0, len(paged.Items))
			for _, dbApp := range paged.Items {
				registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
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
func (a *adapter) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	return entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (app.App, error) {
			dbApp, err := repo.db.App.Query().
				Where(appdb.Namespace(input.Namespace)).
				Where(appdb.ID(input.ID)).
				First(ctx)
			if err != nil {
				if db.IsNotFound(err) {
					return nil, app.NewAppNotFoundError(input)
				}

				return nil, err
			}

			// Get registry item
			registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
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

// UpdateApp updates an app
func (a *adapter) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.App, error) {
		return entutils.TransactingRepo(
			ctx,
			a,
			func(ctx context.Context, repo *adapter) (app.App, error) {
				// Update the app
				_, err := repo.db.App.Update().
					Where(appdb.Namespace(input.AppID.Namespace)).
					Where(appdb.ID(input.AppID.ID)).
					SetName(input.Name).
					SetOrClearDescription(input.Description).
					SetOrClearMetadata(input.Metadata).
					Save(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to update the app with id %s: %w", input.AppID.ID, err)
				}

				// Get the updated app
				app, err := a.GetApp(ctx, input.AppID)
				if err != nil {
					return nil, fmt.Errorf("failed to get updated app: %s: %w", input.AppID.ID, err)
				}

				return app, nil
			})
	})
}

// UninstallApp uninstalls an app
func (a *adapter) UninstallApp(ctx context.Context, input app.UninstallAppInput) (*app.AppBase, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (*app.AppBase, error) {
		return entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (*app.AppBase, error) {
			installedApp, err := repo.GetApp(ctx, input)
			if err != nil {
				return nil, fmt.Errorf("failed to get app: %w", err)
			}

			// Get app factory through registry
			registryItem, err := repo.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
				Type: installedApp.GetType(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get listing for app: %w", err)
			}

			// Uninstall app through factory
			err = registryItem.Factory.UninstallApp(ctx, installedApp.GetID())
			if err != nil {
				return nil, fmt.Errorf("failed to uninstall app: %w", err)
			}

			deletedAt := time.Now()

			// Delete app from database
			_, err = repo.db.App.Update().
				Where(appdb.Namespace(input.Namespace)).
				Where(appdb.ID(input.ID)).
				SetDeletedAt(time.Now()).
				Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to delete app from database: %w", err)
			}

			appBase := installedApp.GetAppBase()
			appBase.DeletedAt = &deletedAt

			return &appBase, nil
		})
	})
}

// mapAppBaseFromDB maps an app base from the database
func mapAppBaseFromDB(dbApp *db.App, registryItem app.RegistryItem) app.AppBase {
	return app.AppBase{
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
func mapAppFromDB(ctx context.Context, dbApp *db.App, registryItem app.RegistryItem) (app.App, error) {
	appBase := mapAppBaseFromDB(dbApp, registryItem)

	app, err := registryItem.Factory.NewApp(ctx, appBase)
	if err != nil {
		return app, fmt.Errorf("failed to create app with %s factory: %w", appBase.Type, err)
	}

	return app, nil
}
