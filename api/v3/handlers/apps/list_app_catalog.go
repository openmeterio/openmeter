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
	ListAppCatalogItemsRequest  = api.ListAppCatalogItemsParams
	ListAppCatalogItemsResponse = api.AppCatalogItemPagePaginatedResponse
	ListAppCatalogItemsHandler  httptransport.HandlerWithArgs[ListAppCatalogItemsRequest, ListAppCatalogItemsResponse, ListAppCatalogItemsRequest]
)

func (h *handler) ListAppCatalogItems() ListAppCatalogItemsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppCatalogItemsRequest) (ListAppCatalogItemsRequest, error) {
			return params, nil
		},
		func(ctx context.Context, request ListAppCatalogItemsRequest) (ListAppCatalogItemsResponse, error) {
			return ListAppCatalogItemsResponse{}, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.JSONResponseEncoder[ListAppCatalogItemsResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-app-catalog-items"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
