package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.MarketplaceService = (*Service)(nil)

func (s *Service) GetListing(ctx context.Context, input app.GetMarketplaceListingInput) (app.MarketplaceListing, error) {
	if err := input.Validate(); err != nil {
		return app.MarketplaceListing{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetListing(ctx, input)
}

func (s *Service) ListListings(ctx context.Context, input app.ListMarketplaceListingInput) (pagination.PagedResponse[app.MarketplaceListing], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[app.MarketplaceListing]{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.ListListings(ctx, input)
}

func (s *Service) InstallAppWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return app.App{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.InstallAppWithAPIKey(ctx, input)
}

func (s *Service) GetOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	if err := input.Validate(); err != nil {
		return app.GetOauth2InstallURLOutput{}, app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.GetOauth2InstallURL(ctx, input)
}

func (s *Service) AuthorizeOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	return s.adapter.AuthorizeOauth2Install(ctx, input)
}
