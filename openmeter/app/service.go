package app

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	MarketplaceService
	AppService
}

type MarketplaceService interface {
	GetListing(ctx context.Context, input GetMarketplaceListingInput) (MarketplaceListing, error)
	ListListings(ctx context.Context, input ListMarketplaceListingInput) (pagination.PagedResponse[MarketplaceListing], error)
	InstallAppWithAPIKey(ctx context.Context, input InstallAppWithAPIKeyInput) (App, error)
	GetOauth2InstallURL(ctx context.Context, input GetOauth2InstallURLInput) (GetOauth2InstallURLOutput, error)
	AuthorizeOauth2Install(ctx context.Context, input AuthorizeOauth2InstallInput) error
}

type AppService interface {
	GetApp(ctx context.Context, input GetAppInput) (App, error)
	ListApps(ctx context.Context, input ListAppInput) (pagination.PagedResponse[App], error)
	UninstallApp(ctx context.Context, input DeleteAppInput) error
}
