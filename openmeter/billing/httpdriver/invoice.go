package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/api"
	apphttpdriver "github.com/openmeterio/openmeter/openmeter/app/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/sortx"
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

				Customers: lo.FromPtr(input.Customers),
				Statuses: lo.Map(
					lo.FromPtr(input.Statuses),
					func(status api.InvoiceStatus, _ int) string {
						return string(status)
					},
				),
				ExtendedStatuses: lo.Map(
					lo.FromPtr(input.ExtendedStatuses),
					func(status string, _ int) billing.InvoiceStatus {
						return billing.InvoiceStatus(status)
					},
				),

				IssuedAfter:   input.IssuedAfter,
				IssuedBefore:  input.IssuedBefore,
				CreatedAfter:  input.CreatedAfter,
				CreatedBefore: input.CreatedBefore,
				Expand:        mapInvoiceExpandToEntity(lo.FromPtr(input.Expand)).SetRecalculateGatheringInvoice(true),

				Order:   sortx.Order(lo.FromPtrOr(input.Order, api.InvoiceOrderByOrderingOrder(sortx.OrderDefault))),
				OrderBy: lo.FromPtr(input.OrderBy),

				IncludeDeleted: lo.FromPtrOr(input.IncludeDeleted, false),

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
				invoice, err := MapInvoiceToAPI(invoice)
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

			pendingLinesFilter := mo.None[[]string]()
			if body.Filters != nil {
				pendingLinesFilter = mo.PointerToOption(body.Filters.LineIds)
			}

			return InvoicePendingLinesActionRequest{
				Customer: customer.CustomerID{
					ID:        body.CustomerId,
					Namespace: ns,
				},

				IncludePendingLines:        pendingLinesFilter,
				AsOf:                       body.AsOf,
				ProgressiveBillingOverride: body.ProgressiveBillingOverride,
			}, nil
		},
		func(ctx context.Context, request InvoicePendingLinesActionRequest) (InvoicePendingLinesActionResponse, error) {
			invoices, err := h.service.InvoicePendingLines(ctx, request)
			if err != nil {
				return nil, err
			}

			out := make([]api.Invoice, 0, len(invoices))

			for _, invoice := range invoices {
				invoice, err := MapInvoiceToAPI(invoice)
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

			if len(params.Expand) == 0 {
				params.Expand = []api.InvoiceExpand{api.InvoiceExpandLines}
			}

			return GetInvoiceRequest{
				Invoice: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				Expand: mapInvoiceExpandToEntity(params.Expand).SetDeletedLines(params.IncludeDeletedLines).SetRecalculateGatheringInvoice(true),
			}, nil
		},
		func(ctx context.Context, request GetInvoiceRequest) (GetInvoiceResponse, error) {
			invoice, err := h.service.GetInvoiceByID(ctx, request)
			if err != nil {
				return GetInvoiceResponse{}, err
			}

			return MapInvoiceToAPI(invoice)
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

			return MapInvoiceToAPI(invoice)
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

type (
	SimulateInvoiceRequest  = billing.SimulateInvoiceInput
	SimulateInvoiceResponse = api.Invoice
	SimulateInvoiceParams   struct {
		CustomerID string
	}
	SimulateInvoiceHandler httptransport.HandlerWithArgs[SimulateInvoiceRequest, SimulateInvoiceResponse, SimulateInvoiceParams]
)

func (h *handler) SimulateInvoice() SimulateInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params SimulateInvoiceParams) (SimulateInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return SimulateInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := api.InvoiceSimulationInput{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return SimulateInvoiceRequest{}, err
			}

			lines, err := slicesx.MapWithErr(body.Lines, mapSimulationLineToEntity)
			if err != nil {
				return SimulateInvoiceRequest{}, billing.ValidationError{
					Err: fmt.Errorf("failed to map simulation lines to entity: %w", err),
				}
			}

			return SimulateInvoiceRequest{
				Namespace:  ns,
				CustomerID: &params.CustomerID,

				Number:   body.Number,
				Currency: currencyx.Code(body.Currency),
				Lines:    billing.NewLineChildren(lines),
			}, nil
		},
		func(ctx context.Context, request SimulateInvoiceRequest) (SimulateInvoiceResponse, error) {
			invoice, err := h.service.SimulateInvoice(ctx, request)
			if err != nil {
				return SimulateInvoiceResponse{}, err
			}

			return MapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[SimulateInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("SimulateInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	UpdateInvoiceRequest struct {
		InvoiceID billing.InvoiceID
		Input     api.InvoiceReplaceUpdate
	}
	UpdateInvoiceResponse = api.Invoice
	UpdateInvoiceParams   struct {
		InvoiceID string
	}
	UpdateInvoiceHandler httptransport.HandlerWithArgs[UpdateInvoiceRequest, UpdateInvoiceResponse, UpdateInvoiceParams]
)

func (h *handler) UpdateInvoice() UpdateInvoiceHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdateInvoiceParams) (UpdateInvoiceRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateInvoiceRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			body := api.InvoiceReplaceUpdate{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateInvoiceRequest{}, err
			}

			return UpdateInvoiceRequest{
				InvoiceID: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				Input: body,
			}, nil
		},
		func(ctx context.Context, request UpdateInvoiceRequest) (UpdateInvoiceResponse, error) {
			invoice, err := h.service.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
				Invoice: request.InvoiceID,
				EditFn: func(invoice *billing.Invoice) error {
					var err error

					invoice.Supplier = mergeInvoiceSupplierFromAPI(invoice.Supplier, request.Input.Supplier)
					invoice.Customer = mergeInvoiceCustomerFromAPI(invoice.Customer, request.Input.Customer)
					invoice.Workflow, err = mergeInvoiceWorkflowFromAPI(invoice.Workflow, request.Input.Workflow)
					if err != nil {
						return err
					}

					invoice.Lines, err = h.mergeInvoiceLinesFromAPI(ctx, invoice, request.Input.Lines)
					if err != nil {
						return err
					}

					// basic fields
					invoice.Description = request.Input.Description
					invoice.Metadata = lo.FromPtrOr(request.Input.Metadata, map[string]string{})
					invoice.DraftUntil = request.Input.DraftUntil

					return nil
				},
			})
			if err != nil {
				return UpdateInvoiceResponse{}, err
			}

			return MapInvoiceToAPI(invoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("UpdateInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func MapInvoiceToAPI(invoice billing.Invoice) (api.Invoice, error) {
	var apps *api.BillingProfileAppsOrReference
	var err error

	if invoice.Workflow.Apps != nil {
		apps, err = mapProfileAppsToAPI(invoice.Workflow.Apps)
		if err != nil {
			return api.Invoice{}, fmt.Errorf("failed to map profile apps to API: %w", err)
		}
	} else {
		apps, err = mapProfileAppReferencesToAPI(&invoice.Workflow.AppReferences)
		if err != nil {
			return api.Invoice{}, fmt.Errorf("failed to map profile app references to API: %w", err)
		}
	}

	out := api.Invoice{
		Id: invoice.ID,

		CreatedAt:            invoice.CreatedAt,
		UpdatedAt:            invoice.UpdatedAt,
		DeletedAt:            invoice.DeletedAt,
		IssuedAt:             invoice.IssuedAt,
		VoidedAt:             invoice.VoidedAt,
		DueAt:                invoice.DueAt,
		DraftUntil:           invoice.DraftUntil,
		SentToCustomerAt:     invoice.SentToCustomerAt,
		QuantitySnapshotedAt: invoice.QuantitySnapshotedAt,
		Period:               mapPeriodToAPI(invoice.Period),

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
		ExternalIds: lo.EmptyableToPtr(api.InvoiceAppExternalIds{
			Invoicing: lo.EmptyableToPtr(invoice.ExternalIDs.Invoicing),
			Payment:   lo.EmptyableToPtr(invoice.ExternalIDs.Payment),
		}),
	}

	out.Workflow = api.InvoiceWorkflowSettings{
		Apps:                   apps,
		SourceBillingProfileId: invoice.Workflow.SourceBillingProfileID,
		Workflow:               mapWorkflowConfigSettingsToAPI(invoice.Workflow.Config),
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

func MapEventInvoiceToAPI(event billing.EventInvoice) (api.Invoice, error) {
	// Prefer the apps from the event
	event.Invoice.Workflow.Apps = nil

	invoice, err := MapInvoiceToAPI(event.Invoice)
	if err != nil {
		return api.Invoice{}, err
	}

	// Let's map the apps, if there are no apps in the event, we will skip generating the profile apps

	apps := api.BillingProfileApps{}

	if event.Apps.Invoicing.Type != "" {
		apps.Invoicing, err = apphttpdriver.MapEventAppToAPI(event.Apps.Invoicing)
		if err != nil {
			return api.Invoice{}, err
		}
	}

	if event.Apps.Payment.Type != "" {
		apps.Payment, err = apphttpdriver.MapEventAppToAPI(event.Apps.Payment)
		if err != nil {
			return api.Invoice{}, err
		}
	}

	if event.Apps.Tax.Type != "" {
		apps.Tax, err = apphttpdriver.MapEventAppToAPI(event.Apps.Tax)
		if err != nil {
			return api.Invoice{}, err
		}
	}

	invoice.Workflow.Apps = &api.BillingProfileAppsOrReference{}
	if err := invoice.Workflow.Apps.FromBillingProfileApps(apps); err != nil {
		return api.Invoice{}, err
	}

	return invoice, nil
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

	out := api.BillingParty{
		Id:   lo.ToPtr(c.CustomerID),
		Name: lo.EmptyableToPtr(c.Name),
	}

	if a != nil {
		out.Addresses = lo.ToPtr([]api.Address{
			{
				Country:     (*string)(a.Country),
				PostalCode:  a.PostalCode,
				State:       a.State,
				City:        a.City,
				Line1:       a.Line1,
				Line2:       a.Line2,
				PhoneNumber: a.PhoneNumber,
			},
		})
	}

	return out
}

func mapInvoiceExpandToEntity(expand []api.InvoiceExpand) billing.InvoiceExpand {
	if len(expand) == 0 {
		return billing.InvoiceExpand{}
	}

	return billing.InvoiceExpand{
		Lines:     slices.Contains(expand, api.InvoiceExpandLines),
		Preceding: slices.Contains(expand, api.InvoiceExpandPreceding),
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

func mergeInvoiceSupplierFromAPI(existing billing.SupplierContact, updatedSupplier api.BillingPartyReplaceUpdate) billing.SupplierContact {
	existing.Name = lo.FromPtr(updatedSupplier.Name)

	if updatedSupplier.Addresses == nil || len(*updatedSupplier.Addresses) == 0 {
		existing.Address = models.Address{}
	} else {
		mappedAddress := customerhttpdriver.MapAddress(&(*updatedSupplier.Addresses)[0])
		existing.Address = *mappedAddress
	}

	if updatedSupplier.TaxId != nil {
		existing.TaxCode = updatedSupplier.TaxId.Code
	} else {
		existing.TaxCode = nil
	}

	return existing
}

func mergeInvoiceCustomerFromAPI(existing billing.InvoiceCustomer, updatedCustomer api.BillingPartyReplaceUpdate) billing.InvoiceCustomer {
	existing.Name = lo.FromPtr(updatedCustomer.Name)

	if updatedCustomer.Addresses == nil || len(*updatedCustomer.Addresses) == 0 {
		existing.BillingAddress = nil
	} else {
		mappedAddress := customerhttpdriver.MapAddress(&(*updatedCustomer.Addresses)[0])
		existing.BillingAddress = mappedAddress
	}

	return existing
}

func mergeInvoiceWorkflowFromAPI(existing billing.InvoiceWorkflow, updatedWorkflow api.InvoiceWorkflowReplaceUpdate) (billing.InvoiceWorkflow, error) {
	existing.Config.Invoicing.AutoAdvance = lo.FromPtrOr(
		updatedWorkflow.Workflow.Invoicing.AutoAdvance,
		billing.DefaultWorkflowConfig.Invoicing.AutoAdvance)

	if updatedWorkflow.Workflow.Invoicing.DraftPeriod == nil {
		existing.Config.Invoicing.DraftPeriod = billing.DefaultWorkflowConfig.Invoicing.DraftPeriod
	} else {
		period, err := isodate.String(*updatedWorkflow.Workflow.Invoicing.DraftPeriod).Parse()
		if err != nil {
			return existing, billing.ValidationError{
				Err: fmt.Errorf("failed to parse draft period: %w", err),
			}
		}

		existing.Config.Invoicing.DraftPeriod = period
	}

	if updatedWorkflow.Workflow.Invoicing.DueAfter == nil {
		existing.Config.Invoicing.DueAfter = billing.DefaultWorkflowConfig.Invoicing.DueAfter
	} else {
		period, err := isodate.String(*updatedWorkflow.Workflow.Invoicing.DueAfter).Parse()
		if err != nil {
			return existing, billing.ValidationError{
				Err: fmt.Errorf("failed to parse due after: %w", err),
			}
		}

		existing.Config.Invoicing.DueAfter = period
	}

	if updatedWorkflow.Workflow.Payment.CollectionMethod != nil {
		existing.Config.Payment.CollectionMethod = billing.CollectionMethod(*updatedWorkflow.Workflow.Payment.CollectionMethod)
	} else {
		existing.Config.Payment.CollectionMethod = billing.DefaultWorkflowConfig.Payment.CollectionMethod
	}

	return existing, nil
}
