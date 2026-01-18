package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3response "github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAppsHandler is a handler for listing apps
type (
	ListAppsRequest  = app.ListAppInput
	ListAppsResponse = apiv3.AppPagePaginatedResponse
	ListAppsParams   = apiv3.ListAppsParams
	ListAppsHandler  httptransport.HandlerWithArgs[ListAppsRequest, ListAppsResponse, ListAppsParams]
)

// ListApps returns a handler for listing apps
func (h *handler) ListApps() ListAppsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListAppsParams) (ListAppsRequest, error) {
			// Resolve namespace
			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListAppsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			return ListAppsRequest{
				Namespace: namespace,
				Page:      page,
			}, nil
		},
		func(ctx context.Context, request ListAppsRequest) (ListAppsResponse, error) {
			result, err := h.appService.ListApps(ctx, request)
			if err != nil {
				return ListAppsResponse{}, fmt.Errorf("failed to list apps: %w", err)
			}

			items := make([]apiv3.BillingApp, 0, len(result.Items))
			for _, item := range result.Items {
				apiAppItem, err := MapAppToAPI(item)
				if err != nil {
					return ListAppsResponse{}, fmt.Errorf("failed to map app to apiv3: %w", err)
				}

				items = append(items, apiAppItem)
			}

			r := apiv3response.NewPagePaginationResponse(items, apiv3response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			})

			response := ConvertToListAppResponse(r)

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListAppsResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("listApps"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
