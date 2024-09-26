package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appdb "github.com/openmeterio/openmeter/openmeter/ent/db/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// ListApps lists apps
func (a adapter) ListApps(ctx context.Context, params app.ListAppInput) (pagination.PagedResponse[app.App], error) {
	db := a.client()

	query := db.App.
		Query().
		Where(appdb.Namespace(params.Namespace))

	// Do not return deleted apps by default
	if !params.IncludeDeleted {
		query = query.Where(appdb.DeletedAtIsNil())
	}

	response := pagination.PagedResponse[app.App]{
		Page: params.Page,
	}

	paged, err := query.Paginate(ctx, params.Page)
	if err != nil {
		return response, err
	}

	result := make([]app.App, 0, len(paged.Items))
	for _, item := range paged.Items {
		result = append(result, *mapAppFromDB(item))
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

// GetApp gets an app
func (a adapter) GetApp(ctx context.Context, input app.GetAppInput) (*app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	dbApp, err := a.client().App.Query().
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

	return mapAppFromDB(dbApp), nil
}

// UninstallApp uninstalls an app
func (a adapter) UninstallApp(ctx context.Context, input app.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// TODO: Implement uninstall logic
	return fmt.Errorf("uninstall not implemented")
}

func mapAppFromDB(dbApp *db.App) *app.App {
	if dbApp == nil {
		return nil
	}

	return &app.App{
		ManagedResource: models.ManagedResource{
			ID: dbApp.ID,
			NamespacedModel: models.NamespacedModel{
				Namespace: dbApp.Namespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: dbApp.CreatedAt,
				UpdatedAt: dbApp.UpdatedAt,
				DeletedAt: dbApp.DeletedAt,
			},
		},
		Type:       dbApp.Type,
		Name:       dbApp.Name,
		Status:     dbApp.Status,
		ListingKey: dbApp.ListingKey,
	}
}
