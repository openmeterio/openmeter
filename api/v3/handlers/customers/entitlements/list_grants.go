package customersentitlements

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomerEntitlementGrantsRequest  = entitlement.ListCustomerEntitlementGrantsInput
	ListCustomerEntitlementGrantsResponse = response.PagePaginationResponse[api.BillingEntitlementGrant]
	ListCustomerEntitlementGrantsParams   struct {
		CustomerID    api.ULID
		EntitlementID api.ULID
		Params        api.ListCustomerEntitlementGrantsParams
	}
	ListCustomerEntitlementGrantsHandler = httptransport.HandlerWithArgs[ListCustomerEntitlementGrantsRequest, ListCustomerEntitlementGrantsResponse, ListCustomerEntitlementGrantsParams]
)

func (h *handler) ListCustomerEntitlementGrants() ListCustomerEntitlementGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerEntitlementGrantsParams) (ListCustomerEntitlementGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementGrantsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Params.Page.Number, 1),
					lo.FromPtrOr(params.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCustomerEntitlementGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			orderBy := grant.OrderByCreatedAt
			order := sortx.OrderAsc
			if params.Params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Params.Sort)
				if err != nil {
					return ListCustomerEntitlementGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}

				orderBy, err = fromAPIEntitlementGrantSortField(ctx, sort.Field)
				if err != nil {
					return ListCustomerEntitlementGrantsRequest{}, err
				}

				order = sort.Order.ToSortxOrder()
			}

			return ListCustomerEntitlementGrantsRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				EntitlementID:  params.EntitlementID,
				IncludeDeleted: lo.FromPtr(params.Params.IncludeDeleted),
				OrderBy:        orderBy,
				Order:          order,
				Page:           page,
			}, nil
		},
		func(ctx context.Context, req ListCustomerEntitlementGrantsRequest) (ListCustomerEntitlementGrantsResponse, error) {
			result, err := h.service.ListCustomerEntitlementGrants(ctx, req)
			if err != nil {
				return ListCustomerEntitlementGrantsResponse{}, err
			}

			now := clock.Now()

			items, err := slicesx.MapWithErr(result.Items, func(g grant.Grant) (api.BillingEntitlementGrant, error) {
				return toAPIEntitlementGrant(g, now)
			})
			if err != nil {
				return ListCustomerEntitlementGrantsResponse{}, fmt.Errorf("converting grants: %w", err)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerEntitlementGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-entitlement-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
