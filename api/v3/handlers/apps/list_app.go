package apps

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListAppsHandler is a handler for listing apps
type (
	ListAppsRequest  = app.ListAppInput
	ListAppsResponse = api.AppPagePaginatedResponse
	ListAppsParams   = api.ListAppsParams
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

			req := ListAppsRequest{
				Namespace: namespace,
				Page:      page,
			}

			if params.Filter != nil {
				id, err := filters.FromAPIFilterULID(params.Filter.Id)
				if err != nil {
					return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.ID = id

				name, err := filters.FromAPIFilterString(params.Filter.Name)
				if err != nil {
					return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[name]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Name = name

				appType, err := filters.FromAPIFilterStringExact(params.Filter.Type)
				if err != nil {
					return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[type]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Type = appType

				status, err := filters.FromAPIFilterStringExact(params.Filter.Status)
				if err != nil {
					return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[status]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Status = status
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListAppsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				req.OrderBy = app.AppOrderBy(sort.Field)
				req.Order = sort.Order.ToSortxOrder()
			}

			return req, nil
		},
		func(ctx context.Context, request ListAppsRequest) (ListAppsResponse, error) {
			result, err := h.appService.ListApps(ctx, request)
			if err != nil {
				return ListAppsResponse{}, fmt.Errorf("failed to list apps: %w", err)
			}

			items, err := ToAPIBillingApps(result.Items)
			if err != nil {
				return ListAppsResponse{}, fmt.Errorf("failed to convert Apps to BillingApps: %w", err)
			}

			r := response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			})

			response := ToAPIAppPagePaginatedResponse(r)

			return response, nil
		},
		commonhttp.JSONResponseEncoder[ListAppsResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-apps"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
