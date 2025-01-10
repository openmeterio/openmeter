package httpdriver

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMarketplaceListingsHandler is a handler for listing marketplace listings
type (
	ListMarketplaceListingsRequest  = appentity.MarketplaceListInput
	ListMarketplaceListingsResponse = api.MarketplaceListingList
	ListMarketplaceListingsParams   = api.ListMarketplaceListingsParams
	ListMarketplaceListingsHandler  httptransport.HandlerWithArgs[ListMarketplaceListingsRequest, ListMarketplaceListingsResponse, ListMarketplaceListingsParams]
)

// ListMarketplaceListings returns a handler for listing marketplace listings
func (h *handler) ListMarketplaceListings() ListMarketplaceListingsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListMarketplaceListingsParams) (ListMarketplaceListingsRequest, error) {
			return ListMarketplaceListingsRequest{
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, app.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, app.DefaultPageNumber),
				},
			}, nil
		},
		func(ctx context.Context, request ListMarketplaceListingsRequest) (ListMarketplaceListingsResponse, error) {
			result, err := h.service.ListMarketplaceListings(ctx, request)
			if err != nil {
				return ListMarketplaceListingsResponse{}, fmt.Errorf("failed to list marketplace listings: %w", err)
			}

			return ListMarketplaceListingsResponse{
				Page:       result.Page.PageNumber,
				PageSize:   result.Page.PageSize,
				TotalCount: result.TotalCount,
				Items: lo.Map(result.Items, func(item appentity.RegistryItem, _ int) api.MarketplaceListing {
					return mapMarketplaceListing(item.Listing)
				}),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMarketplaceListingsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listMarketplaceListings"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// GetMarketplaceListingHandler is a handler to get a marketplace listing
type (
	GetMarketplaceListingRequest  = appentity.MarketplaceGetInput
	GetMarketplaceListingResponse = api.MarketplaceListing
	GetMarketplaceListingHandler  httptransport.HandlerWithArgs[GetMarketplaceListingRequest, GetMarketplaceListingResponse, api.AppType]
)

// GetMarketplaceListing returns a handler for listing marketplace listings
func (h *handler) GetMarketplaceListing() GetMarketplaceListingHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType api.AppType) (GetMarketplaceListingRequest, error) {
			return GetMarketplaceListingRequest{
				Type: appentitybase.AppType(appType),
			}, nil
		},
		func(ctx context.Context, request GetMarketplaceListingRequest) (GetMarketplaceListingResponse, error) {
			result, err := h.service.GetMarketplaceListing(ctx, request)
			if err != nil {
				return GetMarketplaceListingResponse{}, fmt.Errorf("failed to get marketplace listing: %w", err)
			}

			return mapMarketplaceListing(result.Listing), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetMarketplaceListingResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("getMarketplaceListing"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	MarketplaceAppAPIKeyInstallRequest  = appentity.InstallAppWithAPIKeyInput
	MarketplaceAppAPIKeyInstallResponse = api.MarketplaceInstallResponse
	MarketplaceAppAPIKeyInstallHandler  httptransport.HandlerWithArgs[MarketplaceAppAPIKeyInstallRequest, MarketplaceAppAPIKeyInstallResponse, api.AppType]
)

// MarketplaceAppAPIKeyInstall returns a handler for installing an app type with an API key
func (h *handler) MarketplaceAppAPIKeyInstall() MarketplaceAppAPIKeyInstallHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType api.AppType) (MarketplaceAppAPIKeyInstallRequest, error) {
			body := api.MarketplaceAppAPIKeyInstallJSONBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return MarketplaceAppAPIKeyInstallRequest{}, fmt.Errorf("field to decode marketplace app install request: %w", err)
			}

			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return MarketplaceAppAPIKeyInstallRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := MarketplaceAppAPIKeyInstallRequest{
				MarketplaceListingID: appentity.MarketplaceListingID{Type: appentitybase.AppType(appType)},
				Namespace:            namespace,
				APIKey:               body.ApiKey,
				Name:                 lo.FromPtrOr(body.Name, ""),
			}

			return req, nil
		},
		func(ctx context.Context, request MarketplaceAppAPIKeyInstallRequest) (MarketplaceAppAPIKeyInstallResponse, error) {
			resp := MarketplaceAppAPIKeyInstallResponse{
				DefaultForCapabilityTypes: []api.AppCapabilityType{},
			}

			installedApp, err := h.service.InstallMarketplaceListingWithAPIKey(ctx, request)
			if err != nil {
				return resp, err
			}

			// Make stripe the default billing app
			if installedApp.GetType() == appentitybase.AppTypeStripe {
				defaultForCapabilityTypes, err := h.makeStripeDefaultBillingApp(ctx, installedApp)
				if err != nil {
					if errors.As(err, &app.AppProviderPreConditionError{}) {
						// Do not return error if it's a pre-condition error
						// The app is installed successfully but won't be set as the default billing app.
						// The user can set it manually later.
					} else {
						return resp, fmt.Errorf("make stripe app default billing profile: %w", err)
					}
				}

				resp.DefaultForCapabilityTypes = defaultForCapabilityTypes
			}

			// Map app to API
			apiApp, err := h.appMapper.MapAppToAPI(installedApp)
			if err != nil {
				return resp, fmt.Errorf("failed to map app to API: %w", err)
			}

			resp.App = apiApp

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MarketplaceAppAPIKeyInstallResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("marketplaceAppAPIKeyInstall"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// Make Stripe app the default billing app if current one is Sandbox app
func (h *handler) makeStripeDefaultBillingApp(ctx context.Context, app appentity.App) ([]api.AppCapabilityType, error) {
	defaultForCapabilityTypes := []api.AppCapabilityType{}

	appID := app.GetID()

	// Check if it's a Stripe app
	if app.GetType() != appentitybase.AppTypeStripe {
		return defaultForCapabilityTypes, fmt.Errorf("app is not a stripe app: %s", appID.ID)
	}

	// Check if the default billing profile is a sandbox app type
	defaultBillingProfile, err := h.billingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: appID.Namespace,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get default billing profile: %w", err)
	}

	// Do nothing if the default is not the sandbox
	if defaultBillingProfile != nil && defaultBillingProfile.Apps != nil && defaultBillingProfile.Apps.Invoicing.GetType() != appentitybase.AppTypeSandbox {
		return defaultForCapabilityTypes, nil
	}

	// Get supplier contract from stripe app
	supplierContract, err := h.stripeAppService.GetSupplierContact(ctx, appstripeentity.GetSupplierContactInput{
		AppID: appID,
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to get supplier contract for stripe app %s: %w", appID.ID, err)
	}

	// Create new default billing profile
	appRef := billing.AppReference{
		ID: appID.ID,
	}

	_, err = h.billingService.CreateProfile(ctx, billing.CreateProfileInput{
		Namespace:      appID.Namespace,
		Name:           "Stripe Billing Profile",
		Description:    lo.ToPtr("Stripe Billing Profile, created automatically"),
		Default:        true,
		Supplier:       supplierContract,
		WorkflowConfig: billing.DefaultWorkflowConfig,
		Apps: billing.ProfileAppReferences{
			Tax:       appRef,
			Invoicing: appRef,
			Payment:   appRef,
		},
	})
	if err != nil {
		return defaultForCapabilityTypes, fmt.Errorf("failed to create billing profile for stripe app %s: %w", appID.ID, err)
	}

	defaultForCapabilityTypes = []api.AppCapabilityType{
		api.AppCapabilityType(appentitybase.CapabilityTypeCalculateTax),
		api.AppCapabilityType(appentitybase.CapabilityTypeInvoiceCustomers),
		api.AppCapabilityType(appentitybase.CapabilityTypeCollectPayments),
	}

	return defaultForCapabilityTypes, nil
}

func mapMarketplaceListing(listing appentitybase.MarketplaceListing) api.MarketplaceListing {
	return api.MarketplaceListing{
		Type:        api.AppType(listing.Type),
		Name:        listing.Name,
		Description: listing.Description,
		IconUrl:     listing.IconURL,
		Capabilities: lo.Map(listing.Capabilities, func(v appentitybase.Capability, _ int) api.AppCapability {
			return api.AppCapability{
				Type:        api.AppCapabilityType(v.Type),
				Key:         v.Key,
				Name:        v.Name,
				Description: v.Description,
			}
		}),
	}
}
