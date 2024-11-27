package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ InvoiceHandler = (*handler)(nil)

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
					func(status api.BillingInvoiceStatus, _ int) string {
						return string(status)
					},
				),
				ExtendedStatuses: lo.Map(
					lo.FromPtrOr(input.ExtendedStatuses, nil),
					func(status api.BillingInvoiceExtendedStatus, _ int) billingentity.InvoiceStatus {
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

type (
	CreateInvoiceRequest  = billing.CreateInvoiceInput
	CreateInvoiceResponse = []api.BillingInvoice
	CreateInvoiceHandler  httptransport.HandlerWithArgs[CreateInvoiceRequest, CreateInvoiceResponse, string]
)

func (h *handler) CreateInvoice() CreateInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerID string) (CreateInvoiceRequest, error) {
			body := api.BillingCreateInvoiceJSONRequestBody{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateInvoiceRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return CreateInvoiceRequest{
				Customer: customerentity.CustomerID{
					ID:        customerID,
					Namespace: ns,
				},

				IncludePendingLines: mo.PointerToOption(body.IncludePendingLines),
				AsOf:                body.AsOf,
			}, nil
		},
		func(ctx context.Context, request CreateInvoiceRequest) (CreateInvoiceResponse, error) {
			invoices, err := h.service.CreateInvoice(ctx, request)
			if err != nil {
				return nil, err
			}

			out := make([]api.BillingInvoice, 0, len(invoices))

			for _, invoice := range invoices {
				invoice, err := mapInvoiceToAPI(invoice)
				if err != nil {
					return nil, err
				}

				out = append(out, invoice)
			}

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateInvoiceResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingCreateInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetInvoiceRequest  = billing.GetInvoiceByIdInput
	GetInvoiceResponse = api.BillingInvoice
	GetInvoiceParams   struct {
		CustomerID string
		InvoiceID  string
		Expand     []api.BillingInvoiceExpand
	}
	GetInvoiceHandler httptransport.HandlerWithArgs[GetInvoiceRequest, GetInvoiceResponse, GetInvoiceParams]
)

func (h *handler) GetInvoice() GetInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetInvoiceParams) (GetInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}
			if err := h.service.ValidateInvoiceOwnership(ctx, billing.ValidateInvoiceOwnershipInput{
				Namespace:  ns,
				InvoiceID:  params.InvoiceID,
				CustomerID: params.CustomerID,
			}); err != nil {
				return GetInvoiceRequest{}, billingentity.NotFoundError{Err: err}
			}

			return GetInvoiceRequest{
				Invoice: billingentity.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				Expand: mapInvoiceExpandToEntity(params.Expand),
			}, nil
		},
		func(ctx context.Context, request GetInvoiceRequest) (GetInvoiceResponse, error) {
			invoice, err := h.service.GetInvoiceByID(ctx, request)
			if err != nil {
				return GetInvoiceResponse{}, err
			}

			return mapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("billingGetInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type ProgressAction string

const (
	InvoiceProgressActionApprove ProgressAction = "approve"
	InvoiceProgressActionRetry   ProgressAction = "retry"
	InvoiceProgressActionAdvance ProgressAction = "advance"
)

var (
	InvoiceProgressActions = []ProgressAction{
		InvoiceProgressActionApprove,
		InvoiceProgressActionRetry,
		InvoiceProgressActionAdvance,
	}
	invoiceProgressOperationNames = map[ProgressAction]string{
		InvoiceProgressActionApprove: "Approve",
		InvoiceProgressActionRetry:   "Retry",
		InvoiceProgressActionAdvance: "Advance",
	}
)

type (
	ProgressInvoiceRequest struct {
		Invoice billingentity.InvoiceID
	}
	ProgressInvoiceResponse = api.BillingInvoice
	ProgressInvoiceParams   struct {
		CustomerID string
		InvoiceID  string
	}
	ProgressInvoiceHandler httptransport.HandlerWithArgs[ProgressInvoiceRequest, ProgressInvoiceResponse, ProgressInvoiceParams]
)

func (h *handler) ProgressInvoice(action ProgressAction) ProgressInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ProgressInvoiceParams) (ProgressInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ProgressInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			if err := h.service.ValidateInvoiceOwnership(ctx, billing.ValidateInvoiceOwnershipInput{
				Namespace:  ns,
				InvoiceID:  params.InvoiceID,
				CustomerID: params.CustomerID,
			}); err != nil {
				return ProgressInvoiceRequest{}, billingentity.NotFoundError{Err: err}
			}

			if !slices.Contains(InvoiceProgressActions, action) {
				return ProgressInvoiceRequest{}, fmt.Errorf("invalid action: %s", action)
			}

			return ProgressInvoiceRequest{
				Invoice: billingentity.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
			}, nil
		},
		func(ctx context.Context, request ProgressInvoiceRequest) (ProgressInvoiceResponse, error) {
			var invoice billingentity.Invoice
			var err error

			switch action {
			case InvoiceProgressActionApprove:
				invoice, err = h.service.ApproveInvoice(ctx, request.Invoice)
			case InvoiceProgressActionRetry:
				invoice, err = h.service.RetryInvoice(ctx, request.Invoice)
			case InvoiceProgressActionAdvance:
				invoice, err = h.service.AdvanceInvoice(ctx, request.Invoice)
			default:
				return ProgressInvoiceResponse{}, fmt.Errorf("invalid action: %s", action)
			}

			if err != nil {
				return ProgressInvoiceResponse{}, err
			}

			return mapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[ProgressInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName(invoiceProgressOperationNames[action]),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func (h *handler) ConvertListInvoicesByCustomerToListInvoices(customerID string, params api.BillingListInvoicesByCustomerParams) api.BillingListInvoicesParams {
	return api.BillingListInvoicesParams{
		Customers: lo.ToPtr([]string{customerID}),

		Statuses:         params.Statuses,
		ExtendedStatuses: params.ExtendedStatuses,
		IssuedAfter:      params.IssuedAfter,
		IssuedBefore:     params.IssuedBefore,

		Expand: params.Expand,

		Page:     params.Page,
		PageSize: params.PageSize,
		Offset:   params.Offset,
		Limit:    params.Limit,
		Order:    params.Order,
		OrderBy:  params.OrderBy,
	}
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

		CreatedAt:  invoice.CreatedAt,
		UpdatedAt:  invoice.UpdatedAt,
		DeletedAt:  invoice.DeletedAt,
		IssuedAt:   invoice.IssuedAt,
		VoidedAt:   invoice.VoidedAt,
		DueAt:      invoice.DueAt,
		DraftUntil: invoice.DraftUntil,
		Period:     mapPeriodToAPI(invoice.Period),

		Currency: string(invoice.Currency),
		Customer: mapInvoiceCustomerToAPI(invoice.Customer),

		Number:      invoice.Number,
		Description: invoice.Description,
		Metadata:    lo.EmptyableToPtr(invoice.Metadata),

		Status: api.BillingInvoiceStatus(invoice.Status.ShortStatus()),
		StatusDetails: api.BillingInvoiceStatusDetails{
			Failed:         invoice.StatusDetails.Failed,
			Immutable:      invoice.StatusDetails.Immutable,
			ExtendedStatus: api.BillingInvoiceExtendedStatus(invoice.Status),

			AvailableActions: lo.Map(invoice.StatusDetails.AvailableActions, func(a billingentity.InvoiceAction, _ int) api.BillingInvoiceAction {
				return api.BillingInvoiceAction(a)
			}),
		},
		Supplier: mapSupplierContactToAPI(invoice.Supplier),
		Totals:   mapTotalsToAPI(invoice.Totals),
		// TODO[OM-943]: Implement
		Payment: nil,
		Type:    api.BillingInvoiceType(invoice.Type),
		ValidationIssues: lo.EmptyableToPtr(
			lo.Map(invoice.ValidationIssues, func(v billingentity.ValidationIssue, _ int) api.BillingValidationIssue {
				return api.BillingValidationIssue{
					// TODO[later]: CreatedAt, UpdatedAt
					Severity:  api.BillingValidationIssueSeverity(v.Severity),
					Message:   v.Message,
					Code:      lo.EmptyableToPtr(v.Code),
					Component: string(v.Component),
					Field:     lo.EmptyableToPtr(v.Path),
				}
			})),
	}

	if invoice.Workflow != nil {
		out.Workflow = &api.BillingInvoiceWorkflowSettings{
			Apps:                   apps,
			SourceBillingProfileID: invoice.Workflow.SourceBillingProfileID,
			Workflow:               mapWorkflowConfigSettingsToAPI(invoice.Workflow.Config),
			Timezone:               string(invoice.Timezone),
		}
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line *billingentity.Line) (api.BillingInvoiceLine, error) {
		mappedLine, err := mapBillingLineToAPI(line)
		if err != nil {
			return api.BillingInvoiceLine{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
		}

		return mappedLine, nil
	})
	if err != nil {
		return api.BillingInvoice{}, err
	}

	if len(outLines) > 0 {
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

func mapInvoiceExpandToEntity(expand []api.BillingInvoiceExpand) billingentity.InvoiceExpand {
	if len(expand) == 0 {
		return billingentity.InvoiceExpand{}
	}

	if slices.Contains(expand, api.BillingInvoiceExpandAll) {
		return billingentity.InvoiceExpand{
			Lines:        true,
			Preceding:    true,
			WorkflowApps: true,
		}
	}

	return billingentity.InvoiceExpand{
		Lines:        slices.Contains(expand, api.BillingInvoiceExpandLines),
		Preceding:    slices.Contains(expand, api.BillingInvoiceExpandPreceding),
		WorkflowApps: slices.Contains(expand, api.BillingInvoiceExpandWorkflowApps),
	}
}

func mapTotalsToAPI(t billingentity.Totals) api.BillingInvoiceTotals {
	return api.BillingInvoiceTotals{
		Amount:              t.Amount.String(),
		ChargesTotal:        t.ChargesTotal.String(),
		DiscountsTotal:      t.DiscountsTotal.String(),
		TaxesInclusiveTotal: t.TaxesInclusiveTotal.String(),
		TaxesExclusiveTotal: t.TaxesExclusiveTotal.String(),
		TaxesTotal:          t.TaxesTotal.String(),
		Total:               t.Total.String(),
	}
}
