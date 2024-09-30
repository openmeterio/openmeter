package appadapter

import (
	"context"
	"fmt"
	"sync"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
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
	registry map[appentitybase.AppType]appentity.RegistryItem
}

// NewMarketplace creates a new marketplace adapter
func NewMarketplace() *Marketplace {
	return &Marketplace{
		registry: map[appentitybase.AppType]appentity.RegistryItem{},
	}
}

// List lists marketplace listings
func (a Marketplace) List(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error) {
	items := lo.Values(a.registry)
	items = lo.Subset(items, (input.PageNumber-1)*input.PageSize, uint(input.PageSize))

	response := pagination.PagedResponse[appentity.RegistryItem]{
		Page:       input.Page,
		Items:      items,
		TotalCount: len(a.registry),
	}

	return response, nil
}

// Get gets a marketplace listing
func (a Marketplace) Get(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error) {
	if _, ok := a.registry[input.Type]; !ok {
		return appentity.RegistryItem{}, app.MarketplaceListingNotFoundError{
			MarketplaceListingID: input,
		}
	}

	return a.registry[input.Type], nil
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

// Register registers an app type
func (a Marketplace) Register(input appentity.RegisterMarketplaceListingInput) error {
	if _, ok := a.registry[input.Listing.Type]; ok {
		return fmt.Errorf("marketplace listing with key %s already exists", input.Listing.Type)
	}

	if err := input.Listing.Validate(); err != nil {
		return fmt.Errorf("marketplace listing with key %s is invalid: %w", input.Listing.Type, err)
	}

	a.registry[input.Listing.Type] = input

	return nil
}
