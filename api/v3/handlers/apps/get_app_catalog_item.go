package apps

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetAppCatalogItemRequest  = app.AppType
	GetAppCatalogItemResponse = api.BillingAppCatalogItem
	GetAppCatalogItemParams   = api.BillingAppType
	GetAppCatalogItemHandler  httptransport.HandlerWithArgs[GetAppCatalogItemRequest, GetAppCatalogItemResponse, GetAppCatalogItemParams]
)

func (h *handler) GetAppCatalogItem() GetAppCatalogItemHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType GetAppCatalogItemParams) (GetAppCatalogItemRequest, error) {
			return ConvertBillingAppType(appType), nil
		},
		func(ctx context.Context, request GetAppCatalogItemRequest) (GetAppCatalogItemResponse, error) {
			input := app.MarketplaceGetInput{Type: request}
			result, err := h.service.GetMarketplaceListing(ctx, input)
			if err != nil {
				return GetAppCatalogItemResponse{}, fmt.Errorf("failed to get app catalog item: %w", err)
			}

			return ConvertRegistryItem(result), nil
		},
		commonhttp.JSONResponseEncoder[GetAppCatalogItemResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-app-catalog-item"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
