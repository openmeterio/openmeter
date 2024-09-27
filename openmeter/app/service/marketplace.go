package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.MarketplaceService = (*Service)(nil)

func (s *Service) GetListing(ctx context.Context, input appentity.GetMarketplaceListingInput) (appentity.MarketplaceListing, error) {
	if err := input.Validate(); err != nil {
		return appentity.MarketplaceListing{}, app.ValidationError{
			Err: err,
		}
	}

	return s.marketplace.GetListing(ctx, input)
}

func (s *Service) ListListings(ctx context.Context, input appentity.ListMarketplaceListingInput) (pagination.PagedResponse[appentity.MarketplaceListing], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[appentity.MarketplaceListing]{}, app.ValidationError{
			Err: err,
		}
	}

	return s.marketplace.ListListings(ctx, input)
}

func (s *Service) InstallAppWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	if err := input.Validate(); err != nil {
		return nil, app.ValidationError{
			Err: err,
		}
	}

	return s.marketplace.InstallAppWithAPIKey(ctx, input)
}

func (s *Service) GetOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	if err := input.Validate(); err != nil {
		return appentity.GetOauth2InstallURLOutput{}, app.ValidationError{
			Err: err,
		}
	}

	return s.marketplace.GetOauth2InstallURL(ctx, input)
}

func (s *Service) AuthorizeOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return s.marketplace.AuthorizeOauth2Install(ctx, input)
}
