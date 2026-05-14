package meters

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListMetersRequest  = meter.ListMetersParams
	ListMetersResponse = response.PagePaginationResponse[api.Meter]
	ListMetersParams   = api.ListMetersParams
	ListMetersHandler  httptransport.HandlerWithArgs[ListMetersRequest, ListMetersResponse, ListMetersParams]
)

func (h *handler) ListMeters() ListMetersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListMetersParams) (ListMetersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListMetersRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListMetersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			req := ListMetersRequest{
				Namespace: ns,
				Page:      page,
			}

			if params.Filter != nil {
				key, err := filters.FromAPIFilterString(params.Filter.Key)
				if err != nil {
					return ListMetersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[key]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Key = key

				name, err := filters.FromAPIFilterString(params.Filter.Name)
				if err != nil {
					return ListMetersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "filter[name]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.Name = name
			}

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListMetersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				req.OrderBy = meter.OrderBy(sort.Field)
				req.Order = sort.Order.ToSortxOrder()
			}

			return req, nil
		},
		func(ctx context.Context, request ListMetersRequest) (ListMetersResponse, error) {
			resp, err := h.service.ListMeters(ctx, request)
			if err != nil {
				return ListMetersResponse{}, err
			}

			items := lo.Map(resp.Items, func(item meter.Meter, _ int) api.Meter {
				return ToAPIMeter(item)
			})

			r := response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListMetersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-meters"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
