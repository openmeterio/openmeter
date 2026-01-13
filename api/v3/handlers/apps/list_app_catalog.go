package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3response "github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListAppCatalogItemsRequest  = app.MarketplaceListInput
	ListAppCatalogItemsResponse = api.AppCatalogItemPagePaginatedResponse
	ListAppCatalogItemsParams   = api.ListAppCatalogItemsParams
	ListAppCatalogItemsHandler  httptransport.HandlerWithArgs[ListAppCatalogItemsRequest, ListAppCatalogItemsResponse, ListAppCatalogItemsParams]
)

func (h *handler) ListAppCatalogItems() ListAppCatalogItemsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppCatalogItemsParams) (ListAppCatalogItemsRequest, error) {
			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListAppCatalogItemsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			request := ListAppCatalogItemsRequest{
				Page: page,
			}

			return request, nil
		},
		func(ctx context.Context, request ListAppCatalogItemsRequest) (ListAppCatalogItemsResponse, error) {
			result, err := h.service.ListMarketplaceListings(ctx, request)
			if err != nil {
				return ListAppCatalogItemsResponse{}, fmt.Errorf("failed to list app catalog items: %w", err)
			}

			appCatalogItems := lo.Map(result.Items, func(item app.RegistryItem, _ int) api.BillingAppCatalogItem {
				return ConvertRegistryItem(item)
			})

			r := apiv3response.NewPagePaginationResponse(appCatalogItems, apiv3response.PageMetaPage{
				Size:   result.Page.PageSize,
				Number: result.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			})

			response := ConvertListAppCatalogItemsResponse(r)

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListAppCatalogItemsResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-app-catalog-items"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
