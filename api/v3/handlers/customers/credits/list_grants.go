package customerscredits

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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCreditGrantsRequest  = creditgrant.ListInput
	ListCreditGrantsResponse = response.PagePaginationResponse[api.BillingCreditGrant]
	ListCreditGrantsParams   struct {
		CustomerID api.ULID
		Params     api.ListCreditGrantsParams
	}
	ListCreditGrantsHandler httptransport.HandlerWithArgs[ListCreditGrantsRequest, ListCreditGrantsResponse, ListCreditGrantsParams]
)

func (h *handler) ListCreditGrants() ListCreditGrantsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, args ListCreditGrantsParams) (ListCreditGrantsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCreditGrantsRequest{}, err
			}

			req, err := fromAPIListCreditGrantsParams(ctx, args.Params.Page, args.Params.Sort, args.Params.Filter)
			if err != nil {
				return ListCreditGrantsRequest{}, err
			}

			req.Namespace = ns
			req.CustomerIDs = []string{args.CustomerID}

			return req, nil
		},
		h.listCreditGrants,
		commonhttp.JSONResponseEncoderWithStatus[ListCreditGrantsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-credit-grants"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

func (h *handler) listCreditGrants(ctx context.Context, request ListCreditGrantsRequest) (ListCreditGrantsResponse, error) {
	result, err := h.creditGrantService.List(ctx, request)
	if err != nil {
		return ListCreditGrantsResponse{}, fmt.Errorf("list credit grants: %w", err)
	}

	grants, err := slicesx.MapWithErr(result.Items, func(item creditpurchase.Charge) (api.BillingCreditGrant, error) {
		return toAPIBillingCreditGrant(item)
	})
	if err != nil {
		return ListCreditGrantsResponse{}, fmt.Errorf("converting credit grants: %w", err)
	}

	return response.NewPagePaginationResponse(grants, response.PageMetaPage{
		Size:   request.Page.PageSize,
		Number: request.Page.PageNumber,
		Total:  lo.ToPtr(result.TotalCount),
	}), nil
}

// fromAPIListCreditGrantsParams maps the query parameters shared by the per-customer
// and namespace-wide grant listings. Callers set the namespace and customer scope.
func fromAPIListCreditGrantsParams(ctx context.Context, apiPage *api.PagePaginationQuery, apiSort *api.SortQuery, apiFilter *api.ListCreditGrantsParamsFilter) (ListCreditGrantsRequest, error) {
	page := pagination.NewPage(1, 20)
	if apiPage != nil {
		page = pagination.NewPage(
			lo.FromPtrOr(apiPage.Number, 1),
			lo.FromPtrOr(apiPage.Size, 20),
		)
	}

	if err := page.Validate(); err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "page",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req := ListCreditGrantsRequest{
		Page:    page,
		OrderBy: creditpurchase.OrderByCreatedAt,
		Order:   sortx.OrderDesc,
	}

	if apiSort != nil {
		sort, err := request.ParseSortBy(*apiSort)
		if err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
				{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
			})
		}

		req.OrderBy = creditpurchase.OrderBy(sort.Field)
		if err := req.OrderBy.Validate(); err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewUnsupportedSortFieldError(ctx, sort.Field, "created_at", "updated_at", "name", "id")
		}

		req.Order = sort.Order.ToSortxOrder()
	}

	if apiFilter == nil {
		return req, nil
	}

	if apiFilter.Status != nil {
		status, err := fromAPIBillingCreditGrantStatus(*apiFilter.Status)
		if err != nil {
			return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
				{
					Field:  "filter[status]",
					Reason: err.Error(),
					Source: apierrors.InvalidParamSourceQuery,
				},
			})
		}

		req.Status = &status
	}

	if apiFilter.Currency != nil {
		currency := currencyx.Code(*apiFilter.Currency)
		req.Currency = &currency
	}

	key, err := filters.FromAPIFilterString(apiFilter.Key)
	if err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[key]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req.Key = key

	req.UpdatedAt, err = filters.FromAPIFilterDateTime(apiFilter.UpdatedAt)
	if err != nil {
		return ListCreditGrantsRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
			{
				Field:  "filter[updated_at]",
				Reason: err.Error(),
				Source: apierrors.InvalidParamSourceQuery,
			},
		})
	}

	req.FeatureFilters, err = fromAPICreditGrantFeatureFilters(apiFilter.FeatureKey)
	if err != nil {
		return ListCreditGrantsRequest{}, newFeatureKeyFilterBadRequest(ctx, err)
	}

	return req, nil
}
