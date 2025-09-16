package appservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppService = (*Service)(nil)

func (s *Service) RegisterMarketplaceListing(input app.RegisterMarketplaceListingInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return s.adapter.RegisterMarketplaceListing(input)
}

func (s *Service) GetMarketplaceListing(ctx context.Context, input app.MarketplaceGetInput) (app.RegistryItem, error) {
	if err := input.Validate(); err != nil {
		return app.RegistryItem{}, models.NewGenericValidationError(err)
	}

	return s.adapter.GetMarketplaceListing(ctx, input)
}

func (s *Service) ListMarketplaceListings(ctx context.Context, input app.MarketplaceListInput) (pagination.Result[app.RegistryItem], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[app.RegistryItem]{}, models.NewGenericValidationError(err)
	}

	return s.adapter.ListMarketplaceListings(ctx, input)
}

func (s *Service) InstallMarketplaceListingWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.InstallMarketplaceListingWithAPIKey(ctx, input)
}

func (s *Service) InstallMarketplaceListing(ctx context.Context, input app.InstallAppInput) (app.App, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	return s.adapter.InstallMarketplaceListing(ctx, input)
}

func (s *Service) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	if err := input.Validate(); err != nil {
		return app.GetOauth2InstallURLOutput{}, models.NewGenericValidationError(err)
	}

	return s.adapter.GetMarketplaceListingOauth2InstallURL(ctx, input)
}

func (s *Service) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	if err := input.Validate(); err != nil {
		return models.NewGenericValidationError(err)
	}

	return s.adapter.AuthorizeMarketplaceListingOauth2Install(ctx, input)
}
