package apps

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetAppCatalogItemRequest  = api.BillingAppType
	GetAppCatalogItemResponse = api.BillingAppCatalogItem
	GetAppCatalogItemHandler  httptransport.HandlerWithArgs[GetAppCatalogItemRequest, GetAppCatalogItemResponse, GetAppCatalogItemRequest]
)

func (h *handler) GetAppCatalogItem() GetAppCatalogItemHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType GetAppCatalogItemRequest) (GetAppCatalogItemRequest, error) {
			return appType, nil
		},
		func(ctx context.Context, request GetAppCatalogItemRequest) (GetAppCatalogItemResponse, error) {
			return GetAppCatalogItemResponse{}, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.JSONResponseEncoder[GetAppCatalogItemResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-app-catalog-item"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
