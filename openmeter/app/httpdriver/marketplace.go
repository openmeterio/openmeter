package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	MarketplaceAppAPIKeyInstallRequest  = appentity.InstallAppWithAPIKeyInput
	MarketplaceAppAPIKeyInstallResponse = api.AppBase
	MarketplaceAppAPIKeyInstallHandler  httptransport.Handler[MarketplaceAppAPIKeyInstallRequest, MarketplaceAppAPIKeyInstallResponse]
)

// MarketplaceAppAPIKeyInstall returns a handler for installing an app type with an API key
func (h *handler) MarketplaceAppAPIKeyInstall() MarketplaceAppAPIKeyInstallHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (MarketplaceAppAPIKeyInstallRequest, error) {
			body := api.MarketplaceAppAPIKeyInstallJSONBody{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return MarketplaceAppAPIKeyInstallRequest{}, fmt.Errorf("field to decode marketplace app install request: %w", err)
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return MarketplaceAppAPIKeyInstallRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := MarketplaceAppAPIKeyInstallRequest{
				MarketplaceListingID: appentity.MarketplaceListingID{Type: appentitybase.AppType(body.Type)},
				Namespace:            namespace,
				APIKey:               body.ApiKey,
			}

			return req, nil
		},
		func(ctx context.Context, request MarketplaceAppAPIKeyInstallRequest) (MarketplaceAppAPIKeyInstallResponse, error) {
			app, err := h.service.InstallAppWithAPIKey(ctx, request)
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
				Listing: api.MarketplaceListing{
					Type:        api.OpenMeterAppAppType(appBase.Listing.Type),
					Name:        appBase.Listing.Name,
					Description: appBase.Listing.Description,
					IconUrl:     appBase.Listing.IconURL,
					Capabilities: lo.Map(appBase.Listing.Capabilities, func(v appentitybase.Capability, _ int) api.AppCapability {
						return api.AppCapability{
							Type:        api.AppCapabilityType(v.Type),
							Key:         v.Key,
							Name:        v.Name,
							Description: v.Description,
							// TODO(pmarton): adapter to return requirements
							// Requirements: lo.Map(v.Requements, func(v appentitybase.Requirement, _ int) api.AppRequirement {
							// 	return api.AppRequirement{
							// 		Key:         v.Key,
							// 		Name:        v.Name,
							// 		Description: v.Description,
							// 	}
							// }),
						}
					}),
				},
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
