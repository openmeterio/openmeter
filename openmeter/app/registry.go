package app

import (
	"context"
	"errors"
	"sync"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/samber/lo"
)

var (
	createDefaultRegistryOnce sync.Once
	defaultRegistry           *Registry
)

func DefaultRegistry() *Registry {
	createDefaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})

	return defaultRegistry
}

var _ IntegrationRegistryAdapter = (*Registry)(nil)

func NewRegistry() *Registry {
	registry := &Registry{}

	return registry
}

type Registry struct {
	listings map[appentity.AppType]appentity.Integration
}

func (r *Registry) GetListing(ctx context.Context, input appentity.GetMarketplaceListingInput) (appentity.MarketplaceListing, error) {
	listing, ok := r.listings[input.Type]
	if !ok {
		return appentity.MarketplaceListing{}, errors.New("listing not found")
	}

	return listing.Listing, nil
}

func (r *Registry) ListListings(ctx context.Context, input appentity.ListMarketplaceListingInput) (pagination.PagedResponse[appentity.MarketplaceListing], error) {
	items := lo.Values(r.listings)
	items = items[input.PageNumber*input.PageSize : input.PageSize]

	response := pagination.PagedResponse[appentity.MarketplaceListing]{
		Page: input.Page,
		Items: lo.Map[appentity.Integration, appentity.MarketplaceListing](items,
			func(i appentity.Integration, _ int) appentity.MarketplaceListing {
				return i.Listing
			}),
		TotalCount: len(r.listings),
	}

	return response, nil
}

func (r *Registry) RegisterListing(appType appentity.AppType, integration appentity.Integration) {
	r.listings[appType] = integration
}

func (r *Registry) addCapability(appType appentity.AppType, capability appentity.Capability) bool {
	listing, ok := r.listings[appType]
	if !ok {
		return false
	}

	listing.Listing.Capabilities = append(listing.Listing.Capabilities, capability)
	r.listings[appType] = listing
	return true
}

type RegisterDefaultListingsInput struct {
	AppService Service
	DB         entdb.Client
}

func (i RegisterDefaultListingsInput) Validate() error {
	if i.AppService == nil {
		return errors.New("app service is required")
	}

	return nil
}
