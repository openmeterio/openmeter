package app

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	MarketplaceService
	AppService
}

type MarketplaceService interface {
	GetListing(ctx context.Context, input appentity.GetMarketplaceListingInput) (appentity.MarketplaceListing, error)
	ListListings(ctx context.Context, input appentity.ListMarketplaceListingInput) (pagination.PagedResponse[appentity.MarketplaceListing], error)
	InstallAppWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error)
	GetOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error)
	AuthorizeOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error
}

type AppService interface {
	GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error)
	ListApps(ctx context.Context, input appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error)
	UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error
}
