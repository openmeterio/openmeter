package appadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMarketplaceListings lists marketplace listings
func (a adapter) ListMarketplaceListings(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error) {
	items := lo.Values(a.registry)
	items = lo.Subset(items, (input.PageNumber-1)*input.PageSize, uint(input.PageSize))

	response := pagination.PagedResponse[appentity.RegistryItem]{
		Page:       input.Page,
		Items:      items,
		TotalCount: len(a.registry),
	}

	return response, nil
}

// GetMarketplaceListing gets a marketplace listing
func (a adapter) GetMarketplaceListing(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error) {
	if _, ok := a.registry[input.Type]; !ok {
		return appentity.RegistryItem{}, app.MarketplaceListingNotFoundError{
			MarketplaceListingID: input,
		}
	}

	return a.registry[input.Type], nil
}

// InstallMarketplaceListingWithAPIKey installs an app with an API key
func (a adapter) InstallMarketplaceListingWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	// Get registry item
	registryItem, err := a.GetMarketplaceListing(ctx, appentity.MarketplaceGetInput{
		Type: input.Type,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get listing for app %s: %w", input.Type, err)
	}

	// Install app
	app, err := registryItem.Factory.InstallAppWithAPIKey(ctx, appentity.AppFactoryInstallAppWithAPIKeyInput{
		Namespace: input.Namespace,
		APIKey:    input.APIKey,
		BaseURL:   a.baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to install app: %w", err)
	}

	return app, nil
}

// GetMarketplaceListingOauth2InstallURL gets an OAuth2 install URL
func (a adapter) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	return appentity.GetOauth2InstallURLOutput{}, fmt.Errorf("not implemented")
}

// AuthorizeOauth2Install authorizes an OAuth2 install
func (a adapter) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	return fmt.Errorf("not implemented")
}

// RegisterMarketplaceListing registers an app type
func (a adapter) RegisterMarketplaceListing(input appentity.RegisterMarketplaceListingInput) error {
	if _, ok := a.registry[input.Listing.Type]; ok {
		return fmt.Errorf("marketplace listing with key %s already exists", input.Listing.Type)
	}

	if err := input.Listing.Validate(); err != nil {
		return fmt.Errorf("marketplace listing with key %s is invalid: %w", input.Listing.Type, err)
	}

	a.registry[input.Listing.Type] = input

	return nil
}
