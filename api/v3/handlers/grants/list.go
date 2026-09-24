package grants

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	customersentitlements "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlements"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListGrantsRequest  = entitlement.ListNamespaceGrantsInput
	ListGrantsResponse = response.PagePaginationResponse[api.BillingEntitlementGrant]
	ListGrantsParams   = api.ListGrantsParams
	ListGrantsHandler  = httptransport.HandlerWithArgs[ListGrantsRequest, ListGrantsResponse, ListGrantsParams]
)

func (h *handler) ListGrants() ListGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListGrantsParams) (ListGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListGrantsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			orderBy := grant.OrderByCreatedAt
			order := sortx.OrderAsc
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				orderBy, err = customersentitlements.FromAPIEntitlementGrantSortField(ctx, sort.Field)
				if err != nil {
					return ListGrantsRequest{}, err
				}

				order = sort.Order.ToSortxOrder()
			}

			req := ListGrantsRequest{
				Namespace:      ns,
				IncludeDeleted: lo.FromPtr(params.IncludeDeleted),
				OrderBy:        orderBy,
				Order:          order,
				Page:           page,
			}

			if params.Filter != nil {
				req.CustomerID, err = filters.FromAPIFilterULID(params.Filter.CustomerId)
				if err != nil {
					return ListGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[customer_id]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				req.FeatureID, err = filters.FromAPIFilterULID(params.Filter.FeatureId)
				if err != nil {
					return ListGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "filter[feature_id]",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListGrantsRequest) (ListGrantsResponse, error) {
			result, err := h.service.ListNamespaceGrants(ctx, req)
			if err != nil {
				return ListGrantsResponse{}, err
			}

			now := clock.Now()

			items, err := slicesx.MapWithErr(result.Items, func(g grant.Grant) (api.BillingEntitlementGrant, error) {
				return customersentitlements.ToAPIEntitlementGrant(g, now)
			})
			if err != nil {
				return ListGrantsResponse{}, fmt.Errorf("converting grants: %w", err)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
