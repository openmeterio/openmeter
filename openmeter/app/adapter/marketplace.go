package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/samber/lo"
)

var _ app.MarketplaceAdapter = (*adapter)(nil)

// ListListings lists marketplace listings
func (a adapter) ListListings(ctx context.Context, params app.ListMarketplaceListingInput) (pagination.PagedResponse[app.MarketplaceListing], error) {
	response := pagination.PagedResponse[app.MarketplaceListing]{
		Page:  params.Page,
		Items: lo.Values(a.marketplaceListings),
	}

	return response, fmt.Errorf("not implemented")
}

// GetListing gets a marketplace listing
func (a adapter) GetListing(ctx context.Context, input app.GetMarketplaceListingInput) (app.MarketplaceListing, error) {
	if _, ok := a.marketplaceListings[input.Key]; !ok {
		return app.MarketplaceListing{}, app.MarketplaceListingNotFoundError{
			MarketplaceListingID: app.MarketplaceListingID(input),
		}
	}

	return a.marketplaceListings[input.Key], nil
}

// InstallAppWithAPIKey installs an app with an API key
func (a adapter) InstallAppWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	return app.App{}, fmt.Errorf("not implemented")
}

// GetOauth2InstallURL gets an OAuth2 install URL
func (a adapter) GetOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	return app.GetOauth2InstallURLOutput{}, fmt.Errorf("not implemented")
}

// AuthorizeOauth2Install authorizes an OAuth2 install
func (a adapter) AuthorizeOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	return fmt.Errorf("not implemented")
}
