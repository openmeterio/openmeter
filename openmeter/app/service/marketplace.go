package appservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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

func (s *Service) InstallApp(ctx context.Context, input app.InstallAppV3Input) (app.InstallAppV3Output, error) {
	if err := input.Validate(); err != nil {
		return app.InstallAppV3Output{}, models.NewGenericValidationError(err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (app.InstallAppV3Output, error) {
		var installedApp app.App
		var err error
		if input.APIKey != nil {
			installedApp, err = s.adapter.InstallMarketplaceListingWithAPIKey(ctx, app.InstallAppWithAPIKeyInput{
				InstallAppInput: app.InstallAppInput{
					MarketplaceListingID: app.MarketplaceListingID{
						Type: input.Type,
					},
					Namespace: input.Namespace,
					Name:      input.Name,
				},
				APIKey: *input.APIKey,
			})
		} else {
			installedApp, err = s.adapter.InstallMarketplaceListing(ctx, app.InstallAppInput{
				MarketplaceListingID: input.MarketplaceListingID,
				Namespace:            input.Namespace,
				Name:                 input.Name,
			})
		}

		if err != nil {
			return app.InstallAppV3Output{}, err
		}

		out := app.InstallAppV3Output{
			App: installedApp,
		}

		if input.CreateDefaultBillingProfile {
			if input.CreateDefaultBillingProfileFn == nil {
				return app.InstallAppV3Output{}, errors.New("create default billing profile function is required when CreateDefaultBillingProfile is true")
			}
			defaultForCapabilityTypes, err := input.CreateDefaultBillingProfileFn(ctx, installedApp)
			if err != nil {
				return app.InstallAppV3Output{}, fmt.Errorf("create billing profile: %w", err)
			}

			out.DefaultCapabilies = defaultForCapabilityTypes
		}

		return out, nil
	})
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
