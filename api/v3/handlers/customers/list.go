package customers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersResponse = response.PagePaginationResponse[api.BillingCustomer]
	ListCustomersParams   = api.ListCustomersParams
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)

func (h *handler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListCustomersParams) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListCustomersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					apierrors.InvalidParameter{
						Field:  "page",
						Reason: err.Error(),
						Source: apierrors.InvalidParamSourceQuery,
					},
				})
			}

			var orderBy string
			var order sortx.Order
			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListCustomersRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						apierrors.InvalidParameter{
							Field:  "sort",
							Reason: err.Error(),
							Source: apierrors.InvalidParamSourceQuery,
						},
					})
				}
				orderBy = sort.Field
				order = sort.Order.ToSortxOrder()
			}

			var filterKey *string
			if params.Filter != nil {
				if params.Filter.Key != nil {
					filterKey = params.Filter.Key
				}
			}

			req := ListCustomersRequest{
				Namespace: ns,
				Page:      page,
				OrderBy:   orderBy,
				Order:     order,
				Key:       filterKey,
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomersRequest) (ListCustomersResponse, error) {
			resp, err := h.service.ListCustomers(ctx, request)
			if err != nil {
				return ListCustomersResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			customers := lo.Map(resp.Items, func(item customer.Customer, _ int) api.BillingCustomer {
				return ConvertCustomerRequestToBillingCustomer(item)
			})

			r := response.NewPagePaginationResponse(customers, response.PageMetaPage{
				Size:   request.Page.PageSize,
				Number: request.Page.PageNumber,
				Total:  lo.ToPtr(resp.TotalCount),
			})

			return r, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-customers"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
