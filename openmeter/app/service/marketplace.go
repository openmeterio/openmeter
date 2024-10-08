package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) RegisterMarketplaceListing(input appentity.RegisterMarketplaceListingInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.RegisterMarketplaceListing(input)
}

func (s *Service) GetMarketplaceListing(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error) {
	if err := input.Validate(); err != nil {
		return appentity.RegistryItem{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetMarketplaceListing(ctx, input)
}

func (s *Service) ListMarketplaceListings(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[appentity.RegistryItem]{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.ListMarketplaceListings(ctx, input)
}

func (s *Service) InstallMarketplaceListingWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.InstallMarketplaceListingWithAPIKey(ctx, input)
}

func (s *Service) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	if err := input.Validate(); err != nil {
		return appentity.GetOauth2InstallURLOutput{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetMarketplaceListingOauth2InstallURL(ctx, input)
}

func (s *Service) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.AuthorizeMarketplaceListingOauth2Install(ctx, input)
}
