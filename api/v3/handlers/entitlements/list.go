package entitlements

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
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListEntitlementsRequest  = entitlement.ListNamespaceEntitlementsInput
	ListEntitlementsResponse = response.PagePaginationResponse[api.BillingEntitlement]
	ListEntitlementsParams   = api.ListEntitlementsParams
	ListEntitlementsHandler  = httptransport.HandlerWithArgs[ListEntitlementsRequest, ListEntitlementsResponse, ListEntitlementsParams]
)

func (h *handler) ListEntitlements() ListEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListEntitlementsParams) (ListEntitlementsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListEntitlementsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "page", err)
			}

			orderBy := entitlement.ListEntitlementsOrderByCreatedAt
			order := sortx.OrderAsc
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "sort", err)
				}

				orderBy = entitlement.ListEntitlementsOrderBy(sort.Field)
				order = sort.Order.ToSortxOrder()
			}

			req := ListEntitlementsRequest{
				Namespace: ns,
				OrderBy:   orderBy,
				Order:     order,
				Page:      page,
			}

			if params.Filter != nil {
				req.CustomerID, err = filters.FromAPIFilterULID(params.Filter.CustomerId)
				if err != nil {
					return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[customer_id]", err)
				}

				req.FeatureID, err = filters.FromAPIFilterULID(params.Filter.FeatureId)
				if err != nil {
					return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_id]", err)
				}

				req.FeatureKey, err = filters.FromAPIFilterStringExact(params.Filter.FeatureKey)
				if err != nil {
					return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_key]", err)
				}

				req.Type, err = filters.FromAPIFilterStringExact(params.Filter.Type)
				if err != nil {
					return ListEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[type]", err)
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListEntitlementsRequest) (ListEntitlementsResponse, error) {
			result, err := h.service.ListNamespaceEntitlements(ctx, req)
			if err != nil {
				return ListEntitlementsResponse{}, err
			}

			items, err := slicesx.MapWithErr(result.Items, func(ent entitlement.Entitlement) (api.BillingEntitlement, error) {
				return customersentitlements.ToAPIBillingEntitlement(&ent)
			})
			if err != nil {
				return ListEntitlementsResponse{}, fmt.Errorf("converting entitlements: %w", err)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListEntitlementsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-entitlements"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func newInvalidQueryParamError(ctx context.Context, field string, err error) error {
	return apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
		{
			Field:  field,
			Reason: err.Error(),
			Source: apierrors.InvalidParamSourceQuery,
		},
	})
}
