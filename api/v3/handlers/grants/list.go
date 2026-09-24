package grants

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
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
				return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "page", err)
			}

			orderBy := grant.OrderByCreatedAt
			order := sortx.OrderAsc
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "sort", err)
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
				if f := params.Filter.CustomerId; f != nil {
					req.CustomerIDs, err = fromAPIFilterValues(f.Eq, f.Oeq, f.Neq)
					if err != nil {
						return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "filter[customer_id]", err)
					}
				}

				if f := params.Filter.Feature; f != nil {
					req.FeatureIDsOrKeys, err = fromAPIFilterValues(f.Eq, f.Oeq, f.Neq)
					if err != nil {
						return ListGrantsRequest{}, newInvalidQueryParamError(ctx, "filter[feature]", err)
					}
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

// fromAPIFilterValues maps an exact match filter to the values the grant list
// matches any of, which cannot express a negation.
func fromAPIFilterValues(eq *string, oeq []string, neq *string) ([]string, error) {
	switch {
	case neq != nil:
		return nil, errors.New("the neq operator is not supported")
	case eq != nil && len(oeq) > 0:
		return nil, errors.New("only one of eq and oeq can be set")
	case eq != nil:
		return []string{*eq}, nil
	default:
		return oeq, nil
	}
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
