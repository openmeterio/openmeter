package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// MarketplaceListListingsHandler is a handler for listing marketplace listings
type (
	MarketplaceListListingsRequest  = appentity.MarketplaceListInput
	MarketplaceListListingsResponse = api.MarketplaceListingList
	MarketplaceListListingsParams   = api.MarketplaceListListingsParams
	MarketplaceListListingsHandler  httptransport.HandlerWithArgs[MarketplaceListListingsRequest, MarketplaceListListingsResponse, MarketplaceListListingsParams]
)

// MarketplaceListListings returns a handler for listing marketplace listings
func (h *handler) MarketplaceListListings() MarketplaceListListingsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params MarketplaceListListingsParams) (MarketplaceListListingsRequest, error) {
			return MarketplaceListListingsRequest{
				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(params.PageSize, app.DefaultPageSize),
					PageNumber: lo.FromPtrOr(params.Page, app.DefaultPageNumber),
				},
			}, nil
		},
		func(ctx context.Context, request MarketplaceListListingsRequest) (MarketplaceListListingsResponse, error) {
			result, err := h.service.ListMarketplaceListings(ctx, request)
			if err != nil {
				return MarketplaceListListingsResponse{}, fmt.Errorf("failed to list marketplace listings: %w", err)
			}

			return MarketplaceListListingsResponse{
				Page:       result.Page.PageNumber,
				PageSize:   result.Page.PageSize,
				TotalCount: result.TotalCount,
				Items: lo.Map(result.Items, func(item appentity.RegistryItem, _ int) api.MarketplaceListing {
					return mapMarketplaceListing(item.Listing)
				}),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MarketplaceListListingsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("marketplaceListListings"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	MarketplaceAppAPIKeyInstallRequest  = appentity.InstallAppWithAPIKeyInput
	MarketplaceAppAPIKeyInstallResponse = api.AppBase
	MarketplaceAppAPIKeyInstallHandler  httptransport.HandlerWithArgs[MarketplaceAppAPIKeyInstallRequest, MarketplaceAppAPIKeyInstallResponse, api.OpenMeterAppType]
)

// MarketplaceAppAPIKeyInstall returns a handler for installing an app type with an API key
func (h *handler) MarketplaceAppAPIKeyInstall() MarketplaceAppAPIKeyInstallHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType api.OpenMeterAppType) (MarketplaceAppAPIKeyInstallRequest, error) {
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
			}

			return req, nil
		},
		func(ctx context.Context, request MarketplaceAppAPIKeyInstallRequest) (MarketplaceAppAPIKeyInstallResponse, error) {
			app, err := h.service.InstallMarketplaceListingWithAPIKey(ctx, request)
			if err != nil {
				return MarketplaceAppAPIKeyInstallResponse{}, err
			}

			appBase := app.GetAppBase()

			return MarketplaceAppAPIKeyInstallResponse{
				Id:     appBase.ID,
				Name:   appBase.Name,
				Status: api.OpenMeterAppAppStatus(appBase.Status),
				// TODO(pmarton): adapter to implement metadata
				// Metadata: appBase.Metadata,
				Listing:   mapMarketplaceListing(appBase.Listing),
				CreatedAt: appBase.CreatedAt,
				UpdatedAt: appBase.UpdatedAt,
				DeletedAt: appBase.DeletedAt,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[MarketplaceAppAPIKeyInstallResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("marketplaceAppAPIKeyInstall"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapMarketplaceListing(listing appentitybase.MarketplaceListing) api.MarketplaceListing {
	return api.MarketplaceListing{
		Type:        api.OpenMeterAppType(listing.Type),
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