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
	ListInvoicesParams   = api.ListInvoicesParams
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
				Namespaces: []string{ns},

				Customers: lo.FromPtrOr(input.Customers, nil),
				Statuses: lo.Map(
					lo.FromPtrOr(input.Statuses, nil),
					func(status api.InvoiceStatus, _ int) string {
						return string(status)
					},
				),
				ExtendedStatuses: lo.Map(
					lo.FromPtrOr(input.ExtendedStatuses, nil),
					func(status string, _ int) billing.InvoiceStatus {
						return billing.InvoiceStatus(status)
					},
				),

				IssuedAfter:  input.IssuedAfter,
				IssuedBefore: input.IssuedBefore,
				Expand:       mapInvoiceExpandToEntity(lo.FromPtrOr(input.Expand, nil)).SetGatheringTotals(true),

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
				Items:      make([]api.Invoice, 0, len(invoices.Items)),
				Page:       invoices.Page.PageNumber,
				PageSize:   invoices.Page.PageSize,
				TotalCount: invoices.TotalCount,
			}

			for _, invoice := range invoices.Items {
				invoice, err := h.mapInvoiceToAPI(invoice)
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
			httptransport.WithOperationName("ListInvoices"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	InvoicePendingLinesActionRequest  = billing.InvoicePendingLinesInput
	InvoicePendingLinesActionResponse = []api.Invoice
	InvoicePendingLinesActionHandler  httptransport.Handler[InvoicePendingLinesActionRequest, InvoicePendingLinesActionResponse]
)

func (h *handler) InvoicePendingLinesAction() InvoicePendingLinesActionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (InvoicePendingLinesActionRequest, error) {
			body := api.InvoicePendingLinesActionInput{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return InvoicePendingLinesActionRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return InvoicePendingLinesActionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return InvoicePendingLinesActionRequest{
				Customer: customerentity.CustomerID{
					ID:        body.CustomerId,
					Namespace: ns,
				},

				IncludePendingLines: mo.PointerToOption(body.IncludePendingLines),
				AsOf:                body.AsOf,
			}, nil
		},
		func(ctx context.Context, request InvoicePendingLinesActionRequest) (InvoicePendingLinesActionResponse, error) {
			invoices, err := h.service.InvoicePendingLines(ctx, request)
			if err != nil {
				return nil, err
			}

			out := make([]api.Invoice, 0, len(invoices))

			for _, invoice := range invoices {
				invoice, err := h.mapInvoiceToAPI(invoice)
				if err != nil {
					return nil, err
				}

				out = append(out, invoice)
			}

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[InvoicePendingLinesActionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("InvoicePendingLinesAction"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	GetInvoiceRequest  = billing.GetInvoiceByIdInput
	GetInvoiceResponse = api.Invoice
	GetInvoiceParams   struct {
		InvoiceID           string
		Expand              []api.InvoiceExpand
		IncludeDeletedLines bool
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

			return GetInvoiceRequest{
				Invoice: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				Expand: mapInvoiceExpandToEntity(params.Expand).SetDeletedLines(params.IncludeDeletedLines).SetGatheringTotals(true),
			}, nil
		},
		func(ctx context.Context, request GetInvoiceRequest) (GetInvoiceResponse, error) {
			invoice, err := h.service.GetInvoiceByID(ctx, request)
			if err != nil {
				return GetInvoiceResponse{}, err
			}

			return h.mapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("GetInvoice"),
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
		InvoiceProgressActionApprove: "ApproveInvoiceAction",
		InvoiceProgressActionRetry:   "RetryInvoiceAction",
		InvoiceProgressActionAdvance: "AdvanceInvoiceAction",
	}
)

type (
	ProgressInvoiceRequest struct {
		Invoice billing.InvoiceID
	}
	ProgressInvoiceResponse = api.Invoice
	ProgressInvoiceParams   struct {
		InvoiceID string
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

			if !slices.Contains(InvoiceProgressActions, action) {
				return ProgressInvoiceRequest{}, fmt.Errorf("invalid action: %s", action)
			}

			return ProgressInvoiceRequest{
				Invoice: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
			}, nil
		},
		func(ctx context.Context, request ProgressInvoiceRequest) (ProgressInvoiceResponse, error) {
			var invoice billing.Invoice
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

			return h.mapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[ProgressInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName(invoiceProgressOperationNames[action]),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	DeleteInvoiceRequest  = billing.DeleteInvoiceInput
	DeleteInvoiceResponse = struct{}
	DeleteInvoiceParams   struct {
		InvoiceID string
	}
	DeleteInvoiceHandler httptransport.HandlerWithArgs[DeleteInvoiceRequest, DeleteInvoiceResponse, DeleteInvoiceParams]
)

func (h *handler) DeleteInvoice() DeleteInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteInvoiceParams) (DeleteInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return billing.InvoiceID{
				ID:        params.InvoiceID,
				Namespace: ns,
			}, nil
		},
		func(ctx context.Context, request DeleteInvoiceRequest) (DeleteInvoiceResponse, error) {
			if err := h.service.DeleteInvoice(ctx, request); err != nil {
				return DeleteInvoiceResponse{}, err
			}

			return DeleteInvoiceResponse{}, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteInvoiceResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("DeleteInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func (h *handler) mapInvoiceToAPI(invoice billing.Invoice) (api.Invoice, error) {
	var apps *api.BillingProfileAppsOrReference

	// If the workflow is not expanded we won't have this
	if invoice.Workflow != nil {
		var err error

		if invoice.Workflow.Apps != nil {
			apps, err = h.mapProfileAppsToAPI(invoice.Workflow.Apps)
			if err != nil {
				return api.Invoice{}, fmt.Errorf("failed to map profile apps to API: %w", err)
			}
		} else {
			apps, err = mapProfileAppReferencesToAPI(&invoice.Workflow.AppReferences)
			if err != nil {
				return api.Invoice{}, fmt.Errorf("failed to map profile app references to API: %w", err)
			}
		}
	}

	out := api.Invoice{
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

		Status: api.InvoiceStatus(invoice.Status.ShortStatus()),
		StatusDetails: api.InvoiceStatusDetails{
			Failed:         invoice.StatusDetails.Failed,
			Immutable:      invoice.StatusDetails.Immutable,
			ExtendedStatus: string(invoice.Status),

			AvailableActions: mapInvoiceAvailableActionsToAPI(invoice.StatusDetails.AvailableActions),
		},
		Supplier: mapSupplierContactToAPI(invoice.Supplier),
		Totals:   mapTotalsToAPI(invoice.Totals),
		// TODO[OM-943]: Implement
		Payment: nil,
		Type:    api.InvoiceType(invoice.Type),
		ValidationIssues: lo.EmptyableToPtr(
			lo.Map(invoice.ValidationIssues, func(v billing.ValidationIssue, _ int) api.ValidationIssue {
				return api.ValidationIssue{
					Id:        v.ID,
					CreatedAt: v.CreatedAt,
					UpdatedAt: v.UpdatedAt,
					DeletedAt: v.DeletedAt,

					Severity:  api.ValidationIssueSeverity(v.Severity),
					Message:   v.Message,
					Code:      lo.EmptyableToPtr(v.Code),
					Component: string(v.Component),
					Field:     lo.EmptyableToPtr(v.Path),
				}
			})),
		ExternalIDs: lo.EmptyableToPtr(api.InvoiceAppExternalIDs{
			Invoicing: lo.EmptyableToPtr(invoice.ExternalIDs.Invoicing),
			Payment:   lo.EmptyableToPtr(invoice.ExternalIDs.Payment),
		}),
	}

	if invoice.Workflow != nil {
		out.Workflow = &api.InvoiceWorkflowSettings{
			Apps:                   apps,
			SourceBillingProfileID: invoice.Workflow.SourceBillingProfileID,
			Workflow:               mapWorkflowConfigSettingsToAPI(invoice.Workflow.Config),
		}
	}

	outLines, err := slicesx.MapWithErr(invoice.Lines.OrEmpty(), func(line *billing.Line) (api.InvoiceLine, error) {
		mappedLine, err := mapBillingLineToAPI(line)
		if err != nil {
			return api.InvoiceLine{}, fmt.Errorf("failed to map billing line[%s] to API: %w", line.ID, err)
		}

		return mappedLine, nil
	})
	if err != nil {
		return api.Invoice{}, err
	}

	if len(outLines) > 0 {
		out.Lines = &outLines
	}

	return out, nil
}

func mapPeriodToAPI(p *billing.Period) *api.Period {
	if p == nil {
		return nil
	}

	// TODO[later]: let's use a common model for this
	return &api.Period{
		From: p.Start,
		To:   p.End,
	}
}

func mapInvoiceCustomerToAPI(c billing.InvoiceCustomer) api.BillingParty {
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

func mapInvoiceExpandToEntity(expand []api.InvoiceExpand) billing.InvoiceExpand {
	if len(expand) == 0 {
		return billing.InvoiceExpand{}
	}

	if slices.Contains(expand, api.InvoiceExpandAll) {
		return billing.InvoiceExpandAll
	}

	return billing.InvoiceExpand{
		Lines:        slices.Contains(expand, api.InvoiceExpandLines),
		Preceding:    slices.Contains(expand, api.InvoiceExpandPreceding),
		WorkflowApps: slices.Contains(expand, api.InvoiceExpandWorkflowApps),
	}
}

func mapTotalsToAPI(t billing.Totals) api.InvoiceTotals {
	return api.InvoiceTotals{
		Amount:              t.Amount.String(),
		ChargesTotal:        t.ChargesTotal.String(),
		DiscountsTotal:      t.DiscountsTotal.String(),
		TaxesInclusiveTotal: t.TaxesInclusiveTotal.String(),
		TaxesExclusiveTotal: t.TaxesExclusiveTotal.String(),
		TaxesTotal:          t.TaxesTotal.String(),
		Total:               t.Total.String(),
	}
}

func mapInvoiceAvailableActionsToAPI(actions billing.InvoiceAvailableActions) api.InvoiceAvailableActions {
	return api.InvoiceAvailableActions{
		Advance: mapInvoiceAvailableActionDetailsToAPI(actions.Advance),
		Approve: mapInvoiceAvailableActionDetailsToAPI(actions.Approve),
		Delete:  mapInvoiceAvailableActionDetailsToAPI(actions.Delete),
		Retry:   mapInvoiceAvailableActionDetailsToAPI(actions.Retry),
		Void:    mapInvoiceAvailableActionDetailsToAPI(actions.Void),
		Invoice: lo.If(actions.Invoice != nil, &api.InvoiceAvailableActionInvoiceDetails{}).Else(nil),
	}
}

func mapInvoiceAvailableActionDetailsToAPI(actions *billing.InvoiceAvailableActionDetails) *api.InvoiceAvailableActionDetails {
	if actions == nil {
		return nil
	}

	return &api.InvoiceAvailableActionDetails{
		ResultingState: string(actions.ResultingState),
	}
}
