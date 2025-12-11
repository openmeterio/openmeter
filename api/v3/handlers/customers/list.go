package customers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersResponse = response.PagePaginationResponse[Customer]
	ListCustomersHandler  httptransport.Handler[ListCustomersRequest, ListCustomersResponse]
)

func (h *handler) ListCustomers() ListCustomersHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListCustomersRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			attributes, err := request.GetAttributes(r,
				request.WithOffsetPagination(),
				request.WithDefaultSort(&request.SortBy{Field: "name", Order: request.SortOrderAsc}),
			)
			if err != nil {
				return ListCustomersRequest{}, err
			}

			req := ListCustomersRequest{
				Namespace: ns,
				Page:      pagination.NewPage(attributes.Pagination.Number, attributes.Pagination.Size),
			}

			// Pick the first sort if there are multiple
			if len(attributes.Sorts) > 0 {
				req.OrderBy = attributes.Sorts[0].Field
				req.Order = attributes.Sorts[0].Order.ToSortxOrder()
			}

			// Filters
			if attributes.Filters != nil {
				for field, f := range attributes.Filters {
					switch field {
					case "key":
						req.Key = f.ToFilterString()
					case "name":
						req.Name = f.ToFilterString()
					case "primary_email":
						req.PrimaryEmail = f.ToFilterString()
					case "subject":
						req.Subject = f.ToFilterString()
					case "customer_ids":
						req.CustomerIDs = f.ToFilterString()
					}
				}
			}

			return req, nil
		},
		func(ctx context.Context, request ListCustomersRequest) (ListCustomersResponse, error) {
			resp, err := h.service.ListCustomers(ctx, request)
			if err != nil {
				return ListCustomersResponse{}, fmt.Errorf("failed to list customers: %w", err)
			}

			customers := lo.Map(resp.Items, func(item customer.Customer, _ int) Customer {
				return Customer{
					BillingCustomer: ConvertCustomerRequestToBillingCustomer(item),
				}
			})

			// Map the customers to the API
			r := response.NewOffsetPaginationResponse(customers, response.OffsetMetaPage{
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
