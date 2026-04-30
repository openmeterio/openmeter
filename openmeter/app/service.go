package app

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// MarketplaceService manages the in-memory marketplace listing registry and install flows.
type MarketplaceService interface {
	RegisterMarketplaceListing(ctx context.Context, input RegisterMarketplaceListingInput) error
	GetMarketplaceListing(ctx context.Context, input MarketplaceGetInput) (RegistryItem, error)
	ListMarketplaceListings(ctx context.Context, input MarketplaceListInput) (pagination.Result[RegistryItem], error)
	InstallMarketplaceListingWithAPIKey(ctx context.Context, input InstallAppWithAPIKeyInput) (App, error)
	InstallMarketplaceListing(ctx context.Context, input InstallAppInput) (App, error)
	GetMarketplaceListingOauth2InstallURL(ctx context.Context, input GetOauth2InstallURLInput) (GetOauth2InstallURLOutput, error)
	AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input AuthorizeOauth2InstallInput) error
}

// AppLifecycleService manages the lifecycle of installed apps.
type AppLifecycleService interface {
	CreateApp(ctx context.Context, input CreateAppInput) (AppBase, error)
	GetApp(ctx context.Context, input GetAppInput) (App, error)
	UpdateAppStatus(ctx context.Context, input UpdateAppStatusInput) error
	UpdateApp(ctx context.Context, input UpdateAppInput) (App, error)
	ListApps(ctx context.Context, input ListAppInput) (pagination.Result[App], error)
	UninstallApp(ctx context.Context, input UninstallAppInput) error
}

// CustomerDataService manages per-customer data stored by installed apps.
type CustomerDataService interface {
	ListCustomerData(ctx context.Context, input ListCustomerInput) (pagination.Result[CustomerApp], error)
	EnsureCustomer(ctx context.Context, input EnsureCustomerInput) error
	DeleteCustomer(ctx context.Context, input DeleteCustomerInput) error
}

// Service is the full public API for the app domain.
type Service interface {
	MarketplaceService
	AppLifecycleService
	CustomerDataService

	models.ServiceHooks[AppBase]
}

// AppService is an alias kept for backward compatibility; prefer Service.
type AppService = Service
