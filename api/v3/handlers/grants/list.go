package grants

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	customersentitlements "github.com/openmeterio/openmeter/api/v3/handlers/customers/entitlements"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

const defaultListGrantsPageSize = 20

type (
	ListGrantsRequest  = entitlement.ListNamespaceGrantsInput
	ListGrantsResponse = api.EntitlementGrantPaginatedResponse
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

			req := ListGrantsRequest{
				Namespace:      ns,
				IncludeDeleted: lo.FromPtr(params.IncludeDeleted),
				OrderBy:        grant.OrderByCreatedAt,
				Order:          sortx.OrderAsc,
				PageSize:       defaultListGrantsPageSize,
			}

			if params.Page != nil {
				if params.Page.Before != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "page[before]", errors.New("backward pagination is not supported"))
				}

				if params.Page.After != nil {
					req.Cursor, err = pagination.DecodeCursor(*params.Page.After)
					if err != nil {
						return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "page[after]", err)
					}
				}

				req.PageSize = lo.FromPtrOr(params.Page.Size, defaultListGrantsPageSize)
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "sort", err)
				}

				req.OrderBy, err = fromAPIGrantSortField(ctx, sort.Field)
				if err != nil {
					return ListGrantsRequest{}, err
				}

				req.Order = sort.Order.ToSortxOrder()
			}

			if params.Filter != nil {
				req.CustomerID, err = filters.FromAPIFilterULID(params.Filter.CustomerId)
				if err != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "filter[customer_id]", err)
				}

				req.FeatureID, err = filters.FromAPIFilterULID(params.Filter.FeatureId)
				if err != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_id]", err)
				}

				req.FeatureKey, err = filters.FromAPIFilterStringExact(params.Filter.FeatureKey)
				if err != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "filter[feature_key]", err)
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

			meta := api.CursorMeta{
				Page: api.CursorMetaPage{
					Next:     nullable.NewNullNullable[string](),
					Previous: nullable.NewNullNullable[string](),
					Size:     float32(req.PageSize),
				},
			}

			if result.NextCursor != nil {
				meta.Page.Next = nullable.NewNullableWithValue(result.NextCursor.Encode())
			}

			return ListGrantsResponse{
				Data: items,
				Meta: meta,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func fromAPIGrantSortField(ctx context.Context, field string) (grant.OrderBy, error) {
	orderBy := grant.OrderBy(field)
	if !lo.Contains(grant.CursorOrderByValues, orderBy) {
		supported := lo.Map(grant.CursorOrderByValues, func(f grant.OrderBy, _ int) string { return string(f) })

		return "", apierrors.NewUnsupportedSortFieldError(ctx, field, supported...)
	}

	return orderBy, nil
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
