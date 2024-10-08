package app

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	AppAdapter

	entutils.TxCreator
}
type AppAdapter interface {
	// Marketplace
	RegisterMarketplaceListing(input appentity.RegisterMarketplaceListingInput) error
	GetMarketplaceListing(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error)
	ListMarketplaceListings(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error)
	InstallMarketplaceListingWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error)
	GetMarketplaceListingOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error)
	AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error

	// Installed app
	CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error)
	GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error)
	GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error)
	ListApps(ctx context.Context, input appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error)
	UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error
}
