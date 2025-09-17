package app

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	AppAdapter

	entutils.TxCreator
}
type AppAdapter interface {
	// Marketplace
	RegisterMarketplaceListing(input RegisterMarketplaceListingInput) error
	GetMarketplaceListing(ctx context.Context, input MarketplaceGetInput) (RegistryItem, error)
	ListMarketplaceListings(ctx context.Context, input MarketplaceListInput) (pagination.Result[RegistryItem], error)
	InstallMarketplaceListingWithAPIKey(ctx context.Context, input InstallAppWithAPIKeyInput) (App, error)
	InstallMarketplaceListing(ctx context.Context, input InstallAppInput) (App, error)
	GetMarketplaceListingOauth2InstallURL(ctx context.Context, input GetOauth2InstallURLInput) (GetOauth2InstallURLOutput, error)
	AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input AuthorizeOauth2InstallInput) error

	// Installed app
	CreateApp(ctx context.Context, input CreateAppInput) (AppBase, error)
	GetApp(ctx context.Context, input GetAppInput) (App, error)
	UpdateApp(ctx context.Context, input UpdateAppInput) (App, error)
	ListApps(ctx context.Context, input ListAppInput) (pagination.Result[App], error)
	UninstallApp(ctx context.Context, input UninstallAppInput) (*AppBase, error)
	UpdateAppStatus(ctx context.Context, input UpdateAppStatusInput) error

	// Customer data
	ListCustomerData(ctx context.Context, input ListCustomerInput) (pagination.Result[CustomerApp], error)
	EnsureCustomer(ctx context.Context, input EnsureCustomerInput) error
	DeleteCustomer(ctx context.Context, input DeleteCustomerInput) error
}
