package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	GetAppCatalogRequest  = app.GetOauth2InstallURLInput
	GetAppCatalogParam    = apiv3models.BillingAppType
	GetAppCatalogResponse = apiv3models.BillingAppCatalogItem
	GetAppCatalogHandler  httptransport.HandlerWithArgs[GetAppCatalogRequest, GetAppCatalogResponse, GetAppCatalogParam]
)

func (h *handler) GetAppCatalog() GetAppCatalogHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, param GetAppCatalogParam) (GetAppCatalogRequest, error) {
			typ, err := ToDomainAppTypeFromAPIBillingAppType(param)
			if err != nil {
				return GetAppCatalogRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "type",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourcePath,
					},
				})
			}

			return GetAppCatalogRequest{
				Type: typ,
			}, nil
		},
		func(ctx context.Context, request GetAppCatalogRequest) (GetAppCatalogResponse, error) {
			app, err := h.appService.GetMarketplaceListing(ctx, request)
			if err != nil {
				return GetAppCatalogResponse{}, fmt.Errorf("failed to get app catalog: %w", err)
			}

			return ToAPIBillingAppCatalogItem(app.Listing)
		},
		commonhttp.JSONResponseEncoder[GetAppCatalogResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-app-catalog-item"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
