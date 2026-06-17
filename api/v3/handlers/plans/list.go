package plans

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListPlansRequest  = plan.ListPlansInput
	ListPlansResponse = response.PagePaginationResponse[api.BillingPlan]
	ListPlansParams   = api.ListPlansParams
	ListPlansHandler  httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]
)

func (h *handler) ListPlans() ListPlansHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListPlansParams) (ListPlansRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListPlansRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListPlansRequest{
				Namespaces: []string{ns},
				Page:       page,
			}

			if params.Filter != nil {
				key, err := filters.FromAPIFilterString(params.Filter.Key)
				if err != nil {
					return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[key]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Key = key

				name, err := filters.FromAPIFilterString(params.Filter.Name)
				if err != nil {
					return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[name]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Name = name

				currency, err := filters.FromAPIFilterStringExact(params.Filter.Currency)
				if err != nil {
					return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[currency]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Currency = currency

				status, err := filters.FromAPIStatusFilter[productcatalog.PlanStatus](ctx, params.Filter.Status)
				if err != nil {
					return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[status]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Status = status
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListPlansRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.OrderBy = plan.OrderBy(sort.Field)
				req.Order = sort.Order.ToSortxOrder()
			}

			return req, nil
		},
		func(ctx context.Context, req ListPlansRequest) (ListPlansResponse, error) {
			result, err := h.service.ListPlans(ctx, req)
			if err != nil {
				return ListPlansResponse{}, err
			}

			items := make([]api.BillingPlan, 0, len(result.Items))
			for _, p := range result.Items {
				billingPlan, err := ToAPIBillingPlan(p)
				if err != nil {
					return ListPlansResponse{}, err
				}

				items = append(items, billingPlan)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListPlansResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-plans"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
