package invoicesync

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.Service = (*noopAppService)(nil)

type noopAppService struct{}

func (n noopAppService) RegisterMarketplaceListing(input app.RegisterMarketplaceListingInput) error {
	return nil
}

func (n noopAppService) GetMarketplaceListing(ctx context.Context, input app.MarketplaceGetInput) (app.RegistryItem, error) {
	return app.RegistryItem{}, nil
}

func (n noopAppService) ListMarketplaceListings(ctx context.Context, input app.MarketplaceListInput) (pagination.Result[app.RegistryItem], error) {
	return pagination.Result[app.RegistryItem]{}, nil
}

func (n noopAppService) InstallMarketplaceListingWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	return nil, nil
}

func (n noopAppService) InstallMarketplaceListing(ctx context.Context, input app.InstallAppInput) (app.App, error) {
	return nil, nil
}

func (n noopAppService) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	return app.GetOauth2InstallURLOutput{}, nil
}

func (n noopAppService) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	return nil
}

func (n noopAppService) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return app.AppBase{}, nil
}

func (n noopAppService) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
	return nil
}

func (n noopAppService) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	return nil, nil
}

func (n noopAppService) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	return nil, nil
}

func (n noopAppService) ListApps(ctx context.Context, input app.ListAppInput) (pagination.Result[app.App], error) {
	return pagination.Result[app.App]{}, nil
}

func (n noopAppService) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return nil
}

func (n noopAppService) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.Result[app.CustomerApp], error) {
	return pagination.Result[app.CustomerApp]{}, nil
}

func (n noopAppService) EnsureCustomer(ctx context.Context, input app.EnsureCustomerInput) error {
	return nil
}

func (n noopAppService) DeleteCustomer(ctx context.Context, input app.DeleteCustomerInput) error {
	return nil
}
