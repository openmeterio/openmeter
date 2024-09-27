package appadapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var (
	createDefaultMarketplaceOnce sync.Once
	defaultMarketplace           *Marketplace
)

func DefaultMarketplace() *Marketplace {
	createDefaultMarketplaceOnce.Do(func() {
		defaultMarketplace = NewMarketplace()
	})

	return defaultMarketplace
}

var _ app.MarketplaceAdapter = (*Marketplace)(nil)

type Marketplace struct {
	marketplaceListings map[appentity.AppType]appentity.MarketplaceListing
}

// NewMarketplace creates a new marketplace adapter
func NewMarketplace() *Marketplace {
	return &Marketplace{
		marketplaceListings: map[appentity.AppType]appentity.MarketplaceListing{},
	}
}

// ListListings lists marketplace listings
func (a Marketplace) ListListings(ctx context.Context, input appentity.ListMarketplaceListingInput) (pagination.PagedResponse[appentity.MarketplaceListing], error) {
	items := lo.Values(a.marketplaceListings)
	items = items[input.PageNumber*input.PageSize : input.PageSize]

	response := pagination.PagedResponse[appentity.MarketplaceListing]{
		Page:       input.Page,
		Items:      items,
		TotalCount: len(a.marketplaceListings),
	}

	return response, fmt.Errorf("not implemented")
}

// GetListing gets a marketplace listing
func (a Marketplace) GetListing(ctx context.Context, input appentity.GetMarketplaceListingInput) (appentity.MarketplaceListing, error) {
	if _, ok := a.marketplaceListings[input.Type]; !ok {
		return appentity.MarketplaceListing{}, app.MarketplaceListingNotFoundError{
			MarketplaceListingID: input,
		}
	}

	return a.marketplaceListings[input.Type], nil
}

// InstallAppWithAPIKey installs an app with an API key
func (a Marketplace) InstallAppWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	return nil, fmt.Errorf("not implemented")
}

// GetOauth2InstallURL gets an OAuth2 install URL
func (a Marketplace) GetOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	return appentity.GetOauth2InstallURLOutput{}, fmt.Errorf("not implemented")
}

// AuthorizeOauth2Install authorizes an OAuth2 install
func (a Marketplace) AuthorizeOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	return fmt.Errorf("not implemented")
}

// RegisterListing registers a marketplace listing
func (a Marketplace) RegisterListing(listing appentity.RegisterMarketplaceListingInput) error {
	if _, ok := a.marketplaceListings[listing.Type]; ok {
		return fmt.Errorf("marketplace listing with key %s already exists", listing.Type)
	}

	if err := listing.Validate(); err != nil {
		return fmt.Errorf("marketplace listing with key %s is invalid: %w", listing.Type, err)
	}

	a.marketplaceListings[listing.Type] = listing

	return nil
}
