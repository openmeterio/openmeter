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
		listing, err := a.marketplace.GetListing(ctx, app.GetMarketplaceListingInput{
			Type: item.Type,
		})
		if err != nil {
			return response, fmt.Errorf("failed to get listing for app %s: %w", item.ID, err)
		}

		app, err := mapAppFromDB(item, listing)
		if err != nil {
			return response, fmt.Errorf("failed to map app %s: %w", item.ID, err)
		}

		result = append(result, app)
	}

	response.TotalCount = paged.TotalCount
	response.Items = result

	return response, nil
}

// GetApp gets an app
func (a adapter) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
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

	listing, err := a.marketplace.GetListing(ctx, app.GetMarketplaceListingInput{
		Type: dbApp.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", dbApp.ID, err)
	}

	app, err := mapAppFromDB(dbApp, listing)
	if err != nil {
		return nil, fmt.Errorf("failed to map app %s: %w", dbApp.ID, err)
	}

	return app, nil
}

// UninstallApp uninstalls an app
func (a adapter) UninstallApp(ctx context.Context, input app.DeleteAppInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// TODO: Implement uninstall logic
	return fmt.Errorf("uninstall not implemented")
}

func mapAppFromDB(dbApp *db.App, listing app.MarketplaceListing) (app.App, error) {
	if dbApp == nil {
		return nil, fmt.Errorf("app is nil")
	}

	switch dbApp.Type {
	case app.AppTypeStripe:
		stripeApp, err := mapStripeAppFromDB(dbApp, listing)
		if err != nil {
			return nil, fmt.Errorf("failed to map stripe app: %w", err)
		}

		return stripeApp, nil
	default:
		return nil, fmt.Errorf("unsupported app type %s", dbApp.Type)
	}
}

func mapAppBaseFromDB(dbApp *db.App, listing app.MarketplaceListing) app.AppBase {
	return app.AppBase{
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
		Type:    dbApp.Type,
		Name:    dbApp.Name,
		Status:  dbApp.Status,
		Listing: listing,
	}
}

func mapStripeAppFromDB(dbApp *db.App, listing app.MarketplaceListing) (app.StripeApp, error) {
	appBase := mapAppBaseFromDB(dbApp, listing)

	if dbApp.StripeAccountID == nil {
		return app.StripeApp{}, fmt.Errorf("stripe account id is nil")
	}

	if dbApp.StripeLivemode == nil {
		return app.StripeApp{}, fmt.Errorf("stripe livemode is nil")
	}

	return app.StripeApp{
		AppBase:         appBase,
		StripeAccountId: *dbApp.StripeAccountID,
		Livemode:        *dbApp.StripeLivemode,
	}, nil
}
