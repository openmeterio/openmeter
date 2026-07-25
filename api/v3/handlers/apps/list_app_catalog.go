package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	app "github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListAppCatalogRequest  = app.MarketplaceListInput
	ListAppCatalogResponse = api.AppCatalogItemPagePaginatedResponse
	ListAppCatalogParams   = api.ListAppCatalogParams
	ListAppCatalogHandler  httptransport.HandlerWithArgs[ListAppCatalogRequest, ListAppCatalogResponse, ListAppCatalogParams]
)

func (h *handler) ListAppCatalog() ListAppCatalogHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppCatalogParams) (ListAppCatalogRequest, error) {
			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListAppCatalogRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			return ListAppCatalogRequest{
				Page: page,
			}, nil
		},
		func(ctx context.Context, request ListAppCatalogRequest) (ListAppCatalogResponse, error) {
			result, err := h.appService.ListMarketplaceListings(ctx, request)
			if err != nil {
				return ListAppCatalogResponse{}, fmt.Errorf("failed to list apps catalog: %w", err)
			}

			data, err := lo.MapErr(result.Items, func(item app.RegistryItem, _ int) (api.BillingAppCatalogItem, error) {
				return ToAPIBillingAppCatalogItem(item.Listing)
			})
			if err != nil {
				return ListAppCatalogResponse{}, fmt.Errorf("failed to convert apps catalog items: %w", err)
			}

			return ListAppCatalogResponse{
				Data: data,
				Meta: api.PaginatedMeta{
					Page: api.PageMeta{
						Size:   float32(request.Page.PageSize),
						Number: float32(request.Page.PageNumber),
						Total:  float32(result.TotalCount),
					},
				},
			}, nil
		},
		commonhttp.JSONResponseEncoder[ListAppCatalogResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-app-catalog"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
