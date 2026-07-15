package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMarketplaceListingsHandler is a handler for listing marketplace listings
type (
	ListMarketplaceListingsRequest  = app.MarketplaceListInput
	ListMarketplaceListingsResponse = api.MarketplaceListingPaginatedResponse
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
				Items: lo.Map(result.Items, func(item app.RegistryItem, _ int) api.MarketplaceListing {
					return mapMarketplaceListing(item.Listing)
				}),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMarketplaceListingsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listMarketplaceListings"),
		)...,
	)
}

// GetMarketplaceListingHandler is a handler to get a marketplace listing
type (
	GetMarketplaceListingRequest  = app.MarketplaceGetInput
	GetMarketplaceListingResponse = api.MarketplaceListing
	GetMarketplaceListingHandler  httptransport.HandlerWithArgs[GetMarketplaceListingRequest, GetMarketplaceListingResponse, api.AppType]
)

// GetMarketplaceListing returns a handler for listing marketplace listings
func (h *handler) GetMarketplaceListing() GetMarketplaceListingHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType api.AppType) (GetMarketplaceListingRequest, error) {
			return GetMarketplaceListingRequest{
				Type: app.AppType(appType),
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
		)...,
	)
}

type (
	MarketplaceAppAPIKeyInstallResponse = api.MarketplaceInstallResponse
	MarketplaceAppAPIKeyInstallHandler  httptransport.HandlerWithArgs[MarketplaceAppAPIKeyInstallRequest, MarketplaceAppAPIKeyInstallResponse, api.AppType]
)

type MarketplaceAppAPIKeyInstallRequest struct {
	app.InstallAppV3Input
	CreateBillingProfile bool
}

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
				InstallAppV3Input: app.InstallAppV3Input{
					MarketplaceListingID: app.MarketplaceListingID{Type: app.AppType(appType)},
					Namespace:            namespace,
					Name:                 lo.FromPtr(body.Name),
					APIKey:               lo.ToPtr(body.ApiKey),
				},
				CreateBillingProfile: lo.FromPtrOr(body.CreateBillingProfile, true),
			}

			return req, nil
		},
		func(ctx context.Context, request MarketplaceAppAPIKeyInstallRequest) (MarketplaceAppAPIKeyInstallResponse, error) {
			resp := MarketplaceAppAPIKeyInstallResponse{
				DefaultForCapabilityTypes: []api.AppCapabilityType{},
			}

			// Install app
			installedApp, err := h.service.InstallApp(ctx, request.InstallAppV3Input)
			if err != nil {
				return resp, err
			}

			// Map app to API
			apiApp, err := MapAppToAPI(installedApp.App)
			if err != nil {
				return resp, fmt.Errorf("failed to map app to API: %w", err)
			}

			resp.App = apiApp
			resp.DefaultForCapabilityTypes = lo.Map(installedApp.DefaultCapabilies, func(c app.CapabilityType, _ int) api.AppCapabilityType {
				return api.AppCapabilityType(c)
			})

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MarketplaceAppAPIKeyInstallResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("marketplaceAppAPIKeyInstall"),
		)...,
	)
}

type (
	MarketplaceAppInstallResponse = api.MarketplaceInstallResponse
	MarketplaceAppInstallHandler  httptransport.HandlerWithArgs[MarketplaceAppInstallRequest, MarketplaceAppInstallResponse, api.AppType]
)

type MarketplaceAppInstallRequest struct {
	app.InstallAppV3Input
	CreateBillingProfile bool
}

// MarketplaceAppInstall returns a handler for installing an app type
func (h *handler) MarketplaceAppInstall() MarketplaceAppInstallHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType api.AppType) (MarketplaceAppInstallRequest, error) {
			body := api.MarketplaceInstallRequestPayload{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return MarketplaceAppInstallRequest{}, fmt.Errorf("field to decode marketplace app install request: %w", err)
			}

			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return MarketplaceAppInstallRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := MarketplaceAppInstallRequest{
				InstallAppV3Input: app.InstallAppV3Input{
					MarketplaceListingID: app.MarketplaceListingID{Type: app.AppType(appType)},
					Namespace:            namespace,
					Name:                 lo.FromPtr(body.Name),
				},
				CreateBillingProfile: lo.FromPtrOr(body.CreateBillingProfile, true),
			}

			return req, nil
		},
		func(ctx context.Context, request MarketplaceAppInstallRequest) (MarketplaceAppInstallResponse, error) {
			resp := MarketplaceAppInstallResponse{
				DefaultForCapabilityTypes: []api.AppCapabilityType{},
			}

			// Install app
			installedApp, err := h.service.InstallApp(ctx, request.InstallAppV3Input)
			if err != nil {
				return resp, err
			}

			// Map app to API
			apiApp, err := MapAppToAPI(installedApp.App)
			if err != nil {
				return resp, fmt.Errorf("failed to map app to API: %w", err)
			}

			resp.App = apiApp
			resp.DefaultForCapabilityTypes = lo.Map(installedApp.DefaultCapabilies, func(c app.CapabilityType, _ int) api.AppCapabilityType {
				return api.AppCapabilityType(c)
			})

			return resp, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MarketplaceAppInstallResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("marketplaceAppInstall"),
		)...,
	)
}

// Map marketplace listing to API
func mapMarketplaceListing(listing app.MarketplaceListing) api.MarketplaceListing {
	return api.MarketplaceListing{
		Type:        api.AppType(listing.Type),
		Name:        listing.Name,
		Description: listing.Description,
		Capabilities: lo.Map(listing.Capabilities, func(v app.Capability, _ int) api.AppCapability {
			return api.AppCapability{
				Type:        api.AppCapabilityType(v.Type),
				Key:         v.Key,
				Name:        v.Name,
				Description: v.Description,
			}
		}),
		InstallMethods: lo.Map(listing.InstallMethods, func(v app.InstallMethod, _ int) api.InstallMethod {
			return api.InstallMethod(v)
		}),
	}
}
