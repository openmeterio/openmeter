package appadapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMarketplaceListings lists marketplace listings
func (a adapter) ListMarketplaceListings(ctx context.Context, input app.MarketplaceListInput) (pagination.Result[app.RegistryItem], error) {
	items := lo.Values(a.registry)
	items = lo.Subset(items, (input.PageNumber-1)*input.PageSize, uint(input.PageSize))

	response := pagination.Result[app.RegistryItem]{
		Page:       input.Page,
		Items:      items,
		TotalCount: len(a.registry),
	}

	return response, nil
}

// GetMarketplaceListing gets a marketplace listing
func (a adapter) GetMarketplaceListing(ctx context.Context, input app.MarketplaceGetInput) (app.RegistryItem, error) {
	if _, ok := a.registry[input.Type]; !ok {
		return app.RegistryItem{}, models.NewGenericNotFoundError(
			fmt.Errorf("listing with type not found: %s", input.Type),
		)
	}

	return a.registry[input.Type], nil
}

// InstallMarketplaceListingWithAPIKey installs an app with an API key
func (a *adapter) InstallMarketplaceListingWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.App, error) {
		// Get registry item
		registryItem, err := a.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
			Type: input.Type,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get listing for app %s: %w", input.Type, err)
		}

		name, ok := lo.Coalesce(input.Name, registryItem.Listing.Name)
		if !ok {
			return nil, fmt.Errorf("name is required, listing doesn't have a name either")
		}

		installer, ok := registryItem.Factory.(app.AppFactoryInstallWithAPIKey)
		if !ok {
			return nil, models.NewGenericValidationError(fmt.Errorf("app does not support this installation method. Supported methods: %v", registryItem.Listing.InstallMethods))
		}

		// Install app
		app, err := installer.InstallAppWithAPIKey(ctx, app.AppFactoryInstallAppWithAPIKeyInput{
			Namespace: input.Namespace,
			APIKey:    input.APIKey,
			Name:      name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to install app: %w", err)
		}

		return app, nil
	})
}

// InstallMarketplaceListing installs an app
func (a *adapter) InstallMarketplaceListing(ctx context.Context, input app.InstallAppInput) (app.App, error) {
	return transaction.Run(ctx, a, func(ctx context.Context) (app.App, error) {
		// Get registry item
		registryItem, err := a.GetMarketplaceListing(ctx, app.MarketplaceGetInput{
			Type: input.Type,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get listing for app %s: %w", input.Type, err)
		}

		name, ok := lo.Coalesce(input.Name, registryItem.Listing.Name)
		if !ok {
			return nil, fmt.Errorf("name is required, listing doesn't have a name either")
		}

		installer, ok := registryItem.Factory.(app.AppFactoryInstall)
		if !ok {
			return nil, models.NewGenericValidationError(fmt.Errorf("app does not support this installation method. Supported methods: %v", registryItem.Listing.InstallMethods))
		}

		// Install app
		app, err := installer.InstallApp(ctx, app.AppFactoryInstallAppInput{
			Namespace: input.Namespace,
			Name:      name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to install app: %w", err)
		}

		return app, nil
	})
}

// GetMarketplaceListingOauth2InstallURL gets an OAuth2 install URL
func (a adapter) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	return app.GetOauth2InstallURLOutput{}, fmt.Errorf("not implemented")
}

// AuthorizeOauth2Install authorizes an OAuth2 install
func (a adapter) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	return fmt.Errorf("not implemented")
}

// RegisterMarketplaceListing registers an app type
func (a adapter) RegisterMarketplaceListing(input app.RegisterMarketplaceListingInput) error {
	if _, ok := a.registry[input.Listing.Type]; ok {
		return fmt.Errorf("marketplace listing with key %s already exists", input.Listing.Type)
	}

	if err := input.Listing.Validate(); err != nil {
		return fmt.Errorf("marketplace listing with key %s is invalid: %w", input.Listing.Type, err)
	}

	a.registry[input.Listing.Type] = input

	return nil
}
