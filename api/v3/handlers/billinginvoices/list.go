package billinginvoices

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	v3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type (
	ListBillingInvoicesRequest  = billing.ListInvoicesInput
	ListBillingInvoicesResponse = response.PagePaginationResponse[v3.BillingInvoice]
	ListBillingInvoicesParams   = v3.ListInvoicesParams
	ListBillingInvoicesHandler  = httptransport.HandlerWithArgs[ListBillingInvoicesRequest, ListBillingInvoicesResponse, ListBillingInvoicesParams]
)

func (h *handler) ListBillingInvoices() ListBillingInvoicesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListBillingInvoicesParams) (ListBillingInvoicesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListBillingInvoicesRequest{}, err
			}

			page := pagination.NewPage(1, 20)
			if params.Page != nil {
				page = pagination.NewPage(
					lo.FromPtrOr(params.Page.Number, 1),
					lo.FromPtrOr(params.Page.Size, 20),
				)
			}

			if err := page.Validate(); err != nil {
				return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
					{Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
				})
			}

			// Default sort: created_at ascending.
			orderBy := api.InvoiceOrderByCreatedAt
			order := sortx.OrderAsc

			if params.Sort != nil {
				sort, err := request.ParseSortBy(*params.Sort)
				if err != nil {
					return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
						{Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
					})
				}
				orderBy, err = FromAPIInvoiceSortField(ctx, sort.Field)
				if err != nil {
					return ListBillingInvoicesRequest{}, err
				}
				order = sort.Order.ToSortxOrder()
			}

			req := ListBillingInvoicesRequest{
				Namespaces:   []string{ns},
				OnlyStandard: true,
				Page:         page,
				OrderBy:      orderBy,
				Order:        order,
			}

			if params.Filter != nil {
				if params.Filter.Status != nil {
					statuses, err := filters.FromAPIStatusFilter[billing.InvoiceShortStatus](ctx, params.Filter.Status)
					if err != nil {
						return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{Field: "filter[status]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
						})
					}
					req.Statuses = lo.Map(statuses, func(s billing.InvoiceShortStatus, _ int) string { return string(s) })
				}

				if params.Filter.CustomerId != nil {
					customerID, err := filters.FromAPIFilterULID(params.Filter.CustomerId)
					if err != nil {
						return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{Field: "filter[customer_id]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
						})
					}
					req.CustomerID = customerID
				}

				if params.Filter.IssuedAt != nil {
					issuedAt, err := filters.FromAPIFilterDateTime(params.Filter.IssuedAt)
					if err != nil {
						return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{Field: "filter[issued_at]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
						})
					}
					req.IssuedAt = issuedAt
				}

				if params.Filter.ServicePeriodStart != nil {
					periodStart, err := filters.FromAPIFilterDateTime(params.Filter.ServicePeriodStart)
					if err != nil {
						return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{Field: "filter[service_period_start]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
						})
					}
					req.PeriodStart = periodStart
				}

				if params.Filter.CreatedAt != nil {
					createdAt, err := filters.FromAPIFilterDateTime(params.Filter.CreatedAt)
					if err != nil {
						return ListBillingInvoicesRequest{}, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
							{Field: "filter[created_at]", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
						})
					}
					req.CreatedAt = createdAt
				}
			}

			return req, nil
		},
		func(ctx context.Context, req ListBillingInvoicesRequest) (ListBillingInvoicesResponse, error) {
			result, err := h.service.ListInvoices(ctx, req)
			if err != nil {
				return ListBillingInvoicesResponse{}, fmt.Errorf("listing invoices: %w", err)
			}

			items, err := slicesx.MapWithErr(result.Items, func(inv billing.Invoice) (v3.BillingInvoice, error) {
				return ToAPIBillingInvoice(inv)
			})
			if err != nil {
				return ListBillingInvoicesResponse{}, fmt.Errorf("converting invoice: %w", err)
			}

			return response.NewPagePaginationResponse(items, response.PageMetaPage{
				Size:   req.Page.PageSize,
				Number: req.Page.PageNumber,
				Total:  lo.ToPtr(result.TotalCount),
			}), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListBillingInvoicesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("list-invoices"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
