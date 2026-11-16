package customersentitlements

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomerEntitlementsRequest  = entitlement.ListCustomerEntitlementsInput
	ListCustomerEntitlementsResponse = response.PagePaginationResponse[apiv3models.BillingEntitlement]
	ListCustomerEntitlementsParams   struct {
		CustomerID apiv3models.ULID
		Params     apiv3models.ListCustomerEntitlementsParams
	}
	ListCustomerEntitlementsHandler = httptransport.HandlerWithArgs[ListCustomerEntitlementsRequest, ListCustomerEntitlementsResponse, ListCustomerEntitlementsParams]
)

func (h *handler) ListCustomerEntitlements() ListCustomerEntitlementsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomerEntitlementsParams) (ListCustomerEntitlementsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerEntitlementsRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Params.Page.Number, 1),
					lo.FromPtrOr(params.Params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCustomerEntitlementsRequest{}, newInvalidQueryParamError(ctx, "page", err)
			}

			orderBy := entitlement.ListEntitlementsOrderByCreatedAt
			order := sortx.OrderAsc
			if params.Params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Params.Sort)
				if err != nil {
					return ListCustomerEntitlementsRequest{}, newInvalidQueryParamError(ctx, "sort", err)
				}

				orderBy = entitlement.ListEntitlementsOrderBy(sort.Field)
				order = sort.Order.ToSortxOrder()
			}

			req := ListCustomerEntitlementsRequest{
				CustomerID: customer.CustomerID{
					Namespace: ns,
					ID:        params.CustomerID,
				},
				OrderBy: orderBy,
				Order:   order,
				Page:    page,
			}

			if params.Params.Filter != nil {
				req.FeatureID, err = filters.FromAPIFilterULID(params.Params.Filter.FeatureId)
				if err != nil {
					return ListCustomerEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_id]", err)
				}

				req.FeatureKey, err = filters.FromAPIFilterStringExact(params.Params.Filter.FeatureKey)
				if err != nil {
					return ListCustomerEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_key]", err)
				}

				req.Type, err = filters.FromAPIFilterStringExact(params.Params.Filter.Type)
				if err != nil {
					return ListCustomerEntitlementsRequest{}, newInvalidQueryParamError(ctx, "filter[type]", err)
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListCustomerEntitlementsRequest) (ListCustomerEntitlementsResponse, error) {
			result, err := h.service.ListCustomerEntitlements(ctx, req)
			if err != nil {
				return ListCustomerEntitlementsResponse{}, err
			}

			items, err := slicesx.MapWithErr(result.Items, func(ent entitlement.Entitlement) (apiv3models.BillingEntitlement, error) {
				return ToAPIBillingEntitlement(&ent)
			})
			if err != nil {
				return ListCustomerEntitlementsResponse{}, fmt.Errorf("converting entitlements: %w", err)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerEntitlementsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customer-entitlements"),
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
