package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type (
	ListInvoicesRequest  = billing.ListInvoicesInput
	ListInvoicesResponse = api.InvoicePaginatedResponse
	ListInvoicesParams   = api.BillingListInvoicesParams
	ListInvoicesHandler  httptransport.HandlerWithArgs[ListInvoicesRequest, ListInvoicesResponse, ListInvoicesParams]
)

func (h *handler) ListInvoices() ListInvoicesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input ListInvoicesParams) (ListInvoicesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListInvoicesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ListInvoicesRequest{
				Namespace: ns,

				Customers: lo.FromPtrOr(input.Customers, nil),
				Statuses: lo.Map(
					lo.FromPtrOr(input.Statuses, nil),
					func(status api.BillingInvoiceStatus, _ int) billingentity.InvoiceStatus {
						return billingentity.InvoiceStatus(status)
					},
				),

				IssuedAfter:  input.IssuedAfter,
				IssuedBefore: input.IssuedBefore,
				Expand:       mapInvoiceExpandToEntity(lo.FromPtrOr(input.Expand, nil)),

				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(input.PageSize, DefaultPageSize),
					PageNumber: lo.FromPtrOr(input.Page, DefaultPageNumber),
				},
			}, nil
		},
		func(ctx context.Context, request ListInvoicesRequest) (ListInvoicesResponse, error) {
			invoices, err := h.service.ListInvoices(ctx, request)
			if err != nil {
				return ListInvoicesResponse{}, err
			}

			res := ListInvoicesResponse{
				Items:      make([]api.BillingInvoice, 0, len(invoices.Items)),
				Page:       invoices.Page.PageNumber,
				PageSize:   invoices.Page.PageSize,
				TotalCount: invoices.TotalCount,
			}

			for _, invoice := range invoices.Items {
				invoice, err := mapInvoiceToAPI(invoice)
				if err != nil {
					return ListInvoicesResponse{}, err
				}

				res.Items = append(res.Items, invoice)
			}

			return res, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ListInvoicesResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingListLinvoices"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func mapInvoiceToAPI(invoice billingentity.Invoice) (api.BillingInvoice, error) {
	var apps *api.BillingProfileAppsOrReference

	// If the workflow is not expanded we won't have this
	if invoice.Workflow != nil {
		var err error

		if invoice.Workflow.Apps != nil {
			apps, err = mapProfileAppsToAPI(invoice.Workflow.Apps)
			if err != nil {
				return api.BillingInvoice{}, fmt.Errorf("failed to map profile apps to API: %w", err)
			}
		} else {
			apps, err = mapProfileAppReferencesToAPI(&invoice.Workflow.AppReferences)
			if err != nil {
				return api.BillingInvoice{}, fmt.Errorf("failed to map profile app references to API: %w", err)
			}
		}
	}

	out := api.BillingInvoice{
		Id: invoice.ID,

		CreatedAt: invoice.CreatedAt,
		UpdatedAt: invoice.UpdatedAt,
		DeletedAt: invoice.DeletedAt,
		IssuedAt:  invoice.IssuedAt,
		VoidedAt:  invoice.VoidedAt,
		DueAt:     invoice.DueAt,
		Period:    mapPeriodToAPI(invoice.Period),

		Currency: string(invoice.Currency),
		Customer: mapInvoiceCustomerToAPI(invoice.Customer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    lo.EmptyableToPtr(invoice.Metadata),

		Status:   api.BillingInvoiceStatus(invoice.Status),
		Supplier: mapSupplierContactToAPI(invoice.Supplier),
		// TODO[OM-942]: This needs to be (re)implemented
		Totals: api.BillingInvoiceTotals{},
		// TODO[OM-943]: Implement
		Payment: nil,
		Type:    api.BillingInvoiceType(invoice.Type),
	}

	if invoice.Workflow != nil {
		out.Workflow = &api.BillingInvoiceWorkflowSettings{
			Apps:                   apps,
			SourceBillingProfileID: invoice.Workflow.SourceBillingProfileID,
			Workflow:               mapWorkflowConfigSettingsToAPI(invoice.Workflow.WorkflowConfig),
			Timezone:               string(invoice.Timezone),
		}
	}

	if len(invoice.Lines) > 0 {
		outLines := make([]api.BillingInvoiceLine, 0, len(invoice.Lines))

		for _, line := range invoice.Lines {
			mappedLine, err := mapBillingLineToAPI(line)
			if err != nil {
				return api.BillingInvoice{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
			}
			outLines = append(outLines, mappedLine)
		}

		out.Lines = &outLines
	}

	return out, nil
}

func mapPeriodToAPI(p *billingentity.Period) *api.BillingPeriod {
	if p == nil {
		return nil
	}

	return &api.BillingPeriod{
		Start: p.Start,
		End:   p.End,
	}
}

func mapInvoiceCustomerToAPI(c billingentity.InvoiceCustomer) api.BillingParty {
	a := c.BillingAddress

	return api.BillingParty{
		Id:   lo.ToPtr(c.CustomerID),
		Name: lo.EmptyableToPtr(c.Name),
		Addresses: lo.ToPtr([]api.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		}),
	}
}

func mapInvoiceExpandToEntity(expand []api.BillingInvoiceExpand) billing.InvoiceExpand {
	if len(expand) == 0 {
		return billing.InvoiceExpand{}
	}

	if slices.Contains(expand, api.BillingInvoiceExpandAll) {
		return billing.InvoiceExpand{
			Lines:        true,
			Preceding:    true,
			Workflow:     true,
			WorkflowApps: true,
		}
	}

	return billing.InvoiceExpand{
		Lines:        slices.Contains(expand, api.BillingInvoiceExpandLines),
		Preceding:    slices.Contains(expand, api.BillingInvoiceExpandPreceding),
		Workflow:     slices.Contains(expand, api.BillingInvoiceExpandWorkflow),
		WorkflowApps: slices.Contains(expand, api.BillingInvoiceExpandWorkflowApps),
	}
}
