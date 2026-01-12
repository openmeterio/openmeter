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
	GetAppCatalogItemOauth2InstallUrlRequest  = api.BillingAppType
	GetAppCatalogItemOauth2InstallUrlResponse = api.BillingAppCatalogItem
	GetAppCatalogItemOauth2InstallUrlHandler  httptransport.HandlerWithArgs[GetAppCatalogItemOauth2InstallUrlRequest, GetAppCatalogItemOauth2InstallUrlResponse, GetAppCatalogItemOauth2InstallUrlRequest]
)

func (h *handler) GetAppCatalogItemOauth2InstallUrl() GetAppCatalogItemOauth2InstallUrlHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appType GetAppCatalogItemOauth2InstallUrlRequest) (GetAppCatalogItemOauth2InstallUrlRequest, error) {
			return appType, nil
		},
		func(ctx context.Context, request GetAppCatalogItemOauth2InstallUrlRequest) (GetAppCatalogItemOauth2InstallUrlResponse, error) {
			return GetAppCatalogItemOauth2InstallUrlResponse{}, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.JSONResponseEncoder[GetAppCatalogItemOauth2InstallUrlResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-app-catalog-item-oauth2-install-url"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
