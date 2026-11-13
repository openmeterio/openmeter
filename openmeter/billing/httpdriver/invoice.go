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
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
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

			req := ListInvoicesRequest{
				Namespace: ns,

				Statuses: lo.Map(
					lo.FromPtr(input.Statuses),
					func(status api.InvoiceStatus, _ int) string {
						return string(status)
					},
				),
				ExtendedStatuses: lo.Map(
					lo.FromPtr(input.ExtendedStatuses),
					func(status string, _ int) billing.StandardInvoiceStatus {
						return billing.StandardInvoiceStatus(status)
					},
				),

				IssuedAt:    filter.NewFilterTime(input.IssuedAfter, input.IssuedBefore),
				PeriodStart: filter.NewFilterTime(input.PeriodStartAfter, input.PeriodStartBefore),
				CreatedAt:   filter.NewFilterTime(input.CreatedAfter, input.CreatedBefore),

				Expand: mapInvoiceExpandsToEntity(lo.FromPtr(input.Expand)).
					With(billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData),

				Order:   sortx.Order(lo.FromPtrOr(input.Order, api.InvoiceOrderByOrderingOrder(sortx.OrderDefault))),
				OrderBy: lo.FromPtr(input.OrderBy),

				IncludeDeleted: lo.FromPtrOr(input.IncludeDeleted, false),

				Page: pagination.Page{
					PageSize:   lo.FromPtrOr(input.PageSize, DefaultPageSize),
					PageNumber: lo.FromPtrOr(input.Page, DefaultPageNumber),
				},
			}

			if customers := lo.FromPtr(input.Customers); len(customers) > 0 {
				req.CustomerID = &filter.FilterULID{FilterString: filter.FilterString{In: &customers}}
			}

			return req, nil
		},
		func(ctx context.Context, request ListInvoicesRequest) (ListInvoicesResponse, error) {
			// Let's mandate properly set page size and page number.
			if err := request.Page.Validate(); err != nil {
				return ListInvoicesResponse{}, models.NewGenericValidationError(err)
			}

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
				invoice, err := h.MapInvoiceToAPI(ctx, invoice)
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
	InvoicePendingLinesActionRequest struct {
		billing.InvoicePendingLinesInput
		Options []billing.InvoicePendingLinesOption
	}
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

			req := InvoicePendingLinesActionRequest{
				InvoicePendingLinesInput: billing.InvoicePendingLinesInput{
					Customer: customer.CustomerID{
						ID:        body.CustomerId,
						Namespace: ns,
					},
					IncludePendingLines: pendingLinesFilter,
					AsOf:                body.AsOf,
				},
			}

			opts := make([]billing.InvoicePendingLinesOption, 0, 2)
			opts = append(opts, billing.WithBypassCollectionAlignment())
			if body.ProgressiveBillingOverride != nil {
				if *body.ProgressiveBillingOverride {
					opts = append(opts, billing.WithPartialInvoiceLinesEnabled())
				} else {
					opts = append(opts, billing.WithPartialInvoiceLinesDisabled())
				}
			}

			req.Options = opts

			return req, nil
		},
		func(ctx context.Context, request InvoicePendingLinesActionRequest) (InvoicePendingLinesActionResponse, error) {
			invoices, err := h.service.InvoicePendingLines(ctx, request.InvoicePendingLinesInput, request.Options...)
			if err != nil {
				return nil, err
			}

			out := make([]api.Invoice, 0, len(invoices))

			for _, invoice := range invoices {
				invoice, err := MapStandardInvoiceToAPI(invoice)
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
				Expand: mapInvoiceExpandsToEntity(params.Expand).
					SetOrUnsetIf(params.IncludeDeletedLines, billing.InvoiceExpandDeletedLines).
					With(billing.InvoiceExpandCalculateGatheringInvoiceWithLiveData),
			}, nil
		},
		func(ctx context.Context, request GetInvoiceRequest) (GetInvoiceResponse, error) {
			invoice, err := h.service.GetInvoiceById(ctx, request)
			if err != nil {
				return GetInvoiceResponse{}, err
			}

			return h.MapInvoiceToAPI(ctx, invoice)
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
	InvoiceProgressActionApprove            ProgressAction = "approve"
	InvoiceProgressActionRetry              ProgressAction = "retry"
	InvoiceProgressActionAdvance            ProgressAction = "advance"
	InvoiceProgressActionSnapshotQuantities ProgressAction = "snapshot_quantities"
)

var (
	InvoiceProgressActions = []ProgressAction{
		InvoiceProgressActionApprove,
		InvoiceProgressActionRetry,
		InvoiceProgressActionAdvance,
		InvoiceProgressActionSnapshotQuantities,
	}
	invoiceProgressOperationNames = map[ProgressAction]string{
		InvoiceProgressActionApprove:            "ApproveInvoiceAction",
		InvoiceProgressActionRetry:              "RetryInvoiceAction",
		InvoiceProgressActionAdvance:            "AdvanceInvoiceAction",
		InvoiceProgressActionSnapshotQuantities: "SnapshotQuantitiesAction",
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
			var invoice billing.StandardInvoice
			var err error

			switch action {
			case InvoiceProgressActionApprove:
				invoice, err = h.service.ApproveInvoice(ctx, request.Invoice)
			case InvoiceProgressActionRetry:
				invoice, err = h.service.RetryInvoice(ctx, request.Invoice)
			case InvoiceProgressActionAdvance:
				invoice, err = h.service.AdvanceInvoice(ctx, request.Invoice)
			case InvoiceProgressActionSnapshotQuantities:
				invoice, err = h.service.ForceCollectInvoice(ctx, request.Invoice)
			default:
				return ProgressInvoiceResponse{}, fmt.Errorf("invalid action: %s", action)
			}

			if err != nil {
				return ProgressInvoiceResponse{}, err
			}

			return MapStandardInvoiceToAPI(invoice)
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

			return billing.DeleteInvoiceInput{
				Invoice: billing.InvoiceID{
					ID:        params.InvoiceID,
					Namespace: ns,
				},
				DeletionSource: billing.ChangeSourceAPIRequest,
			}, nil
		},
		func(ctx context.Context, request DeleteInvoiceRequest) (DeleteInvoiceResponse, error) {
			invoice, err := h.service.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: request.Invoice,
				Expand:  billing.InvoiceExpandAll,
			})
			if err != nil {
				return DeleteInvoiceResponse{}, err
			}

			switch invoice.Type() {
			case billing.InvoiceTypeGathering:
				if err := billing.ValidateAPIInvoiceDeleteSupported(invoice); err != nil {
					return DeleteInvoiceResponse{}, err
				}

				if _, err := h.service.DeleteGatheringInvoice(ctx, request); err != nil {
					return DeleteInvoiceResponse{}, fmt.Errorf("deleting gathering invoice: %w", err)
				}
			case billing.InvoiceTypeStandard:
				if err := h.deleteStandardInvoice(ctx, request); err != nil {
					return DeleteInvoiceResponse{}, fmt.Errorf("deleting standard invoice: %w", err)
				}
			default:
				return DeleteInvoiceResponse{}, models.NewNillableGenericValidationError(fmt.Errorf("invalid invoice type: %s", invoice.Type()))
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

func (h *handler) deleteStandardInvoice(ctx context.Context, request DeleteInvoiceRequest) error {
	invoice, err := h.service.DeleteInvoice(ctx, request)
	if err != nil {
		return err
	}

	// Given we are doing background processing, we might be in any delete.* state, but in case we ended up in delete.failed let's have
	// proper return code for the API (otherwise we would return 200)
	if invoice.Status == billing.StandardInvoiceStatusDeleteFailed {
		// If we have validation issues we return them as the deletion sync handler
		// yields validation errors
		if len(invoice.ValidationIssues) > 0 {
			return billing.ValidationError{
				Err: invoice.ValidationIssues.AsError(),
			}
		}

		return billing.ValidationError{
			Err: fmt.Errorf("%w [status=%s]", billing.ErrInvoiceDeleteFailed, invoice.Status),
		}
	}

	return nil
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
				Currency: currencyx.FiatCode(body.Currency),
				Lines:    billing.NewStandardInvoiceLines(lines),
			}, nil
		},
		func(ctx context.Context, request SimulateInvoiceRequest) (SimulateInvoiceResponse, error) {
			invoice, err := h.service.SimulateInvoice(ctx, request)
			if err != nil {
				return SimulateInvoiceResponse{}, err
			}

			return MapStandardInvoiceToAPI(invoice)
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
			// This fetch is only used to dispatch by invoice type. We do not need a customer lock here:
			// invoice customer IDs are immutable, and the typed update services lock and refetch before editing.
			invoice, err := h.service.GetInvoiceById(ctx, billing.GetInvoiceByIdInput{
				Invoice: request.InvoiceID,
			})
			if err != nil {
				return UpdateInvoiceResponse{}, err
			}

			var updatedInvoice billing.Invoice

			switch invoice.Type() {
			case billing.InvoiceTypeGathering:
				updatedInvoice, err = h.updateGatheringInvoice(ctx, invoice, request)
				if err != nil {
					h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
					return UpdateInvoiceResponse{}, fmt.Errorf("updating gathering invoice: %w", err)
				}
			case billing.InvoiceTypeStandard:
				updatedInvoice, err = h.updateStandardInvoice(ctx, invoice, request)
				if err != nil {
					h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
					return UpdateInvoiceResponse{}, fmt.Errorf("updating standard invoice: %w", err)
				}
			default:
				return UpdateInvoiceResponse{}, models.NewNillableGenericValidationError(fmt.Errorf("invalid invoice type: %s", invoice.Type()))
			}

			return h.MapInvoiceToAPI(ctx, updatedInvoice)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("UpdateInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

func (h *handler) logInvoiceEditRejected(ctx context.Context, invoiceType billing.InvoiceType, request UpdateInvoiceRequest, err error) {
	if h.logger == nil {
		return
	}

	h.logger.WarnContext(ctx, "billing invoice edit rejected",
		"namespace", request.InvoiceID.Namespace,
		"invoiceID", request.InvoiceID.ID,
		"invoiceType", invoiceType,
		// Line edits contain billing configuration but no invoice, customer, or supplier PII.
		"lines", request.Input.Lines,
		"rejection", err.Error(),
	)
}

func (h *handler) updateGatheringInvoice(ctx context.Context, invoice billing.Invoice, request UpdateInvoiceRequest) (billing.Invoice, error) {
	gatheringInvoice, err := invoice.AsGatheringInvoice()
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("converting invoice to gathering invoice: %w", err)
	}

	updatedGatheringInvoice, err := h.service.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
		Invoice:      gatheringInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.GatheringInvoice) error {
			var err error
			invoice.Lines, err = h.mergeGatheringInvoiceLinesFromAPI(ctx, invoice, request.Input.Lines)
			if err != nil {
				return fmt.Errorf("merging lines: %w", err)
			}

			return nil
		},
	})
	if err != nil {
		return billing.Invoice{}, err
	}

	return billing.NewInvoice(updatedGatheringInvoice), nil
}

func (h *handler) updateStandardInvoice(ctx context.Context, invoice billing.Invoice, request UpdateInvoiceRequest) (billing.Invoice, error) {
	stdInvoice, err := invoice.AsStandardInvoice()
	if err != nil {
		return billing.Invoice{}, fmt.Errorf("converting invoice to standard invoice: %w", err)
	}

	updatedStandardInvoice, err := h.service.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
		Invoice:      stdInvoice.GetInvoiceID(),
		ChangeSource: billing.ChangeSourceAPIRequest,
		EditFn: func(invoice *billing.StandardInvoice) error {
			invoice.Supplier = mergeInvoiceSupplierFromAPI(invoice.Supplier, request.Input.Supplier)
			invoice.Customer = mergeInvoiceCustomerFromAPI(invoice.Customer, request.Input.Customer)

			var err error
			invoice.Workflow, err = mergeInvoiceWorkflowFromAPI(invoice.Workflow, request.Input.Workflow)
			if err != nil {
				return fmt.Errorf("merging workflow: %w", err)
			}

			invoice.Lines, err = h.mergeStandardInvoiceLinesFromAPI(ctx, invoice, request.Input.Lines)
			if err != nil {
				return fmt.Errorf("merging lines: %w", err)
			}

			// basic fields
			invoice.Description = request.Input.Description
			invoice.Metadata = lo.FromPtrOr(request.Input.Metadata, map[string]string{})

			return nil
		},
	})
	if err != nil {
		return billing.Invoice{}, err
	}

	return billing.NewInvoice(updatedStandardInvoice), nil
}

func (h *handler) MapInvoiceToAPI(ctx context.Context, invoice billing.Invoice) (api.Invoice, error) {
	switch invoice.Type() {
	case billing.InvoiceTypeStandard:
		standardInvoice, err := invoice.AsStandardInvoice()
		if err != nil {
			return api.Invoice{}, fmt.Errorf("converting invoice to standard invoice: %w", err)
		}

		return MapStandardInvoiceToAPI(standardInvoice)
	case billing.InvoiceTypeGathering:
		gatheringInvoice, err := invoice.AsGatheringInvoice()
		if err != nil {
			return api.Invoice{}, fmt.Errorf("converting invoice to gathering invoice: %w", err)
		}

		// TODO: For the V3 api let's make sure that we don't return gathering invoice customer data (or even gathering invoices)
		mergedProfile, err := h.service.GetCustomerBillingProfile(ctx, billing.GetCustomerBillingProfileInput{
			Customer: gatheringInvoice.GetCustomerID(),
			Expand: billing.CustomerBillingProfileExpand{
				Customer: true,
				Apps:     true,
			},
		})
		if err != nil {
			return UpdateInvoiceResponse{}, fmt.Errorf("failed to get customer override: %w", err)
		}

		return MapGatheringInvoiceToAPI(gatheringInvoice, mergedProfile.Customer, mergedProfile.EffectiveProfile)
	default:
		return api.Invoice{}, fmt.Errorf("invalid invoice type: %s", invoice.Type())
	}
}

func MapStandardInvoiceToAPI(invoice billing.StandardInvoice) (api.Invoice, error) {
	return billingconvert.ToAPIStandardInvoice(invoice)
}

func MapEventInvoiceToAPI(event billing.EventStandardInvoice) (api.Invoice, error) {
	return billingconvert.ToAPIEventInvoice(event)
}

func mapInvoiceExpandsToEntity(expand []api.InvoiceExpand) billing.InvoiceExpands {
	if len(expand) == 0 {
		return nil
	}

	return billing.InvoiceExpands{}.
		SetOrUnsetIf(slices.Contains(expand, api.InvoiceExpandLines), billing.InvoiceExpandLines)
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
		period, err := datetime.ISODurationString(*updatedWorkflow.Workflow.Invoicing.DraftPeriod).Parse()
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
		period, err := datetime.ISODurationString(*updatedWorkflow.Workflow.Invoicing.DueAfter).Parse()
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
