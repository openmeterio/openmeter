package httpdriver

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingconvert "github.com/openmeterio/openmeter/openmeter/billing/apiconvert"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerhttpdriver "github.com/openmeterio/openmeter/openmeter/customer/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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
	ListInvoicesResponse = apiv1models.InvoicePaginatedResponse
	ListInvoicesParams   = apiv1models.ListInvoicesParams
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
					func(status apiv1models.InvoiceStatus, _ int) string {
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

				Order:   sortx.Order(lo.FromPtrOr(input.Order, apiv1models.InvoiceOrderByOrderingOrder(sortx.OrderDefault))),
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
				Items:      make([]apiv1models.Invoice, 0, len(invoices.Items)),
				Page:       invoices.Page.PageNumber,
				PageSize:   invoices.Page.PageSize,
				TotalCount: invoices.TotalCount,
			}

			items, err := h.mapInvoicesToAPI(ctx, request.Namespace, invoices.Items...)
			if err != nil {
				return ListInvoicesResponse{}, err
			}

			res.Items = append(res.Items, items...)

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
	InvoicePendingLinesActionResponse = []apiv1models.Invoice
	InvoicePendingLinesActionHandler  httptransport.Handler[InvoicePendingLinesActionRequest, InvoicePendingLinesActionResponse]
)

func (h *handler) InvoicePendingLinesAction() InvoicePendingLinesActionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (InvoicePendingLinesActionRequest, error) {
			body := apiv1models.InvoicePendingLinesActionInput{}

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

			out := make([]apiv1models.Invoice, 0, len(invoices))

			taxCodes, err := h.taxCodesForStandardInvoices(ctx, request.Customer.Namespace, invoices...)
			if err != nil {
				return nil, err
			}

			for _, invoice := range invoices {
				invoice, err := MapStandardInvoiceToAPI(invoice, taxCodes)
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
	GetInvoiceResponse = apiv1models.Invoice
	GetInvoiceParams   struct {
		InvoiceID           string
		Expand              []apiv1models.InvoiceExpand
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
				params.Expand = []apiv1models.InvoiceExpand{apiv1models.InvoiceExpandLines}
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
	ProgressInvoiceResponse = apiv1models.Invoice
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

			return h.mapStandardInvoiceWithTaxCodesToAPI(ctx, invoice)
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
	SimulateInvoiceRequest struct {
		Input billing.SimulateInvoiceInput
		// UnresolvedStripeCodes holds, by line ID, the stripe.code of lines whose code no Tax
		// Code maps to yet. Simulation can't create the Tax Code, so these lines are rendered
		// with the requested code instead of the invoice default they inherit.
		UnresolvedStripeCodes map[string]string
	}
	SimulateInvoiceResponse = apiv1models.Invoice
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

			body := apiv1models.InvoiceSimulationInput{}

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return SimulateInvoiceRequest{}, err
			}

			unresolvedByIndex, err := resolveSimulationLineTaxConfigs(ctx, h.taxCodeResolver, ns, body.Lines)
			if err != nil {
				return SimulateInvoiceRequest{}, err
			}

			lines, err := slicesx.MapWithErr(body.Lines, mapSimulationLineToEntity)
			if err != nil {
				return SimulateInvoiceRequest{}, billing.ValidationError{
					Err: fmt.Errorf("failed to map simulation lines to entity: %w", err),
				}
			}

			// The simulation keeps the line IDs it is given, so an ID is how the rendered line
			// is matched back to its unresolved code.
			unresolvedStripeCodes := make(map[string]string, len(unresolvedByIndex))
			for idx, stripeCode := range unresolvedByIndex {
				if lines[idx].ID == "" {
					lines[idx].ID = ulid.Make().String()
				}

				unresolvedStripeCodes[lines[idx].ID] = stripeCode
			}

			return SimulateInvoiceRequest{
				Input: billing.SimulateInvoiceInput{
					Namespace:  ns,
					CustomerID: &params.CustomerID,

					Number:   body.Number,
					Currency: currencyx.FiatCode(body.Currency),
					Lines:    billing.NewStandardInvoiceLines(lines),
				},
				UnresolvedStripeCodes: unresolvedStripeCodes,
			}, nil
		},
		func(ctx context.Context, request SimulateInvoiceRequest) (SimulateInvoiceResponse, error) {
			invoice, err := h.service.SimulateInvoice(ctx, request.Input)
			if err != nil {
				return SimulateInvoiceResponse{}, err
			}

			out, err := h.mapStandardInvoiceWithTaxCodesToAPI(ctx, invoice)
			if err != nil {
				return SimulateInvoiceResponse{}, err
			}

			renderUnresolvedSimulationStripeCodes(&out, request.UnresolvedStripeCodes)

			return out, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[SimulateInvoiceResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("SimulateInvoice"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

// renderUnresolvedSimulationStripeCodes shows the requested stripe.code on simulated lines whose
// code no Tax Code maps to yet. Those lines carry no Tax Code ID, so the calculation filled in
// the invoice default's; creating the line would instead create a Tax Code for the requested
// code, so the preview shows that code and no Tax Code ID. The tax behavior stays as calculated.
func renderUnresolvedSimulationStripeCodes(invoice *apiv1models.Invoice, unresolvedStripeCodes map[string]string) {
	if len(unresolvedStripeCodes) == 0 || invoice.Lines == nil {
		return
	}

	for i := range *invoice.Lines {
		line := &(*invoice.Lines)[i]

		stripeCode, ok := unresolvedStripeCodes[line.Id]
		if !ok {
			continue
		}

		taxConfig := &apiv1models.TaxConfig{
			Stripe: &apiv1models.StripeTaxConfig{Code: stripeCode},
		}
		if line.TaxConfig != nil {
			taxConfig.Behavior = line.TaxConfig.Behavior
		}

		line.TaxConfig = taxConfig
		if line.RateCard != nil {
			line.RateCard.TaxConfig = taxConfig
		}

		for j := range lo.FromPtr(line.Children) {
			child := &(*line.Children)[j]
			child.TaxConfig = taxConfig
			if child.RateCard != nil {
				child.RateCard.TaxConfig = taxConfig
			}
		}
	}
}

type (
	UpdateInvoiceRequest struct {
		InvoiceID billing.InvoiceID
		Input     apiv1models.InvoiceReplaceUpdate
	}
	UpdateInvoiceResponse = apiv1models.Invoice
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

			body := apiv1models.InvoiceReplaceUpdate{}

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

			switch invoice.Type() {
			case billing.InvoiceTypeGathering:
				resp, err := h.updateGatheringInvoice(ctx, invoice, request)
				if err != nil {
					return UpdateInvoiceResponse{}, fmt.Errorf("updating gathering invoice: %w", err)
				}

				return resp, nil
			case billing.InvoiceTypeStandard:
				resp, err := h.updateStandardInvoice(ctx, invoice, request)
				if err != nil {
					return UpdateInvoiceResponse{}, fmt.Errorf("updating standard invoice: %w", err)
				}

				return resp, nil
			default:
				return UpdateInvoiceResponse{}, models.NewNillableGenericValidationError(fmt.Errorf("invalid invoice type: %s", invoice.Type()))
			}
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

func (h *handler) updateGatheringInvoice(ctx context.Context, invoice billing.Invoice, request UpdateInvoiceRequest) (UpdateInvoiceResponse, error) {
	gatheringInvoice, err := invoice.AsGatheringInvoice()
	if err != nil {
		return UpdateInvoiceResponse{}, fmt.Errorf("converting invoice to gathering invoice: %w", err)
	}

	// Resolution, the edit and the response share one transaction, so a stripe.code is created
	// with (or rolled back with) the edit, and a failed response rolls the edit back too. The
	// EditFn only applies the already-resolved IDs.
	var resp UpdateInvoiceResponse
	if err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
		resolvedTaxConfigs, err := resolveUpdateLineTaxConfigs(ctx, resolver, request.InvoiceID.Namespace, request.Input.Lines)
		if err != nil {
			h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
			return err
		}

		result, err := h.service.UpdateGatheringInvoice(ctx, billing.UpdateGatheringInvoiceInput{
			Invoice:      gatheringInvoice.GetInvoiceID(),
			ChangeSource: billing.ChangeSourceAPIRequest,
			EditFn: func(invoice *billing.GatheringInvoice) error {
				var err error
				invoice.Lines, err = h.mergeGatheringInvoiceLinesFromAPI(ctx, invoice, request.Input.Lines, resolvedTaxConfigs)
				if err != nil {
					return fmt.Errorf("merging lines: %w", err)
				}

				return nil
			},
		})
		if err != nil {
			h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
			return err
		}

		resp, err = h.MapInvoiceToAPI(ctx, billing.NewInvoice(result))

		return err
	}); err != nil {
		return UpdateInvoiceResponse{}, err
	}

	return resp, nil
}

func (h *handler) updateStandardInvoice(ctx context.Context, invoice billing.Invoice, request UpdateInvoiceRequest) (UpdateInvoiceResponse, error) {
	stdInvoice, err := invoice.AsStandardInvoice()
	if err != nil {
		return UpdateInvoiceResponse{}, fmt.Errorf("converting invoice to standard invoice: %w", err)
	}

	// Same transaction scope as updateGatheringInvoice.
	var resp UpdateInvoiceResponse
	if err := h.taxCodeResolver.WithResolver(ctx, func(ctx context.Context, resolver productcatalog.TaxCodeBackfillResolver) error {
		resolvedTaxConfigs, err := resolveUpdateLineTaxConfigs(ctx, resolver, request.InvoiceID.Namespace, request.Input.Lines)
		if err != nil {
			h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
			return err
		}

		result, err := h.service.UpdateStandardInvoice(ctx, billing.UpdateStandardInvoiceInput{
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

				invoice.Lines, err = h.mergeStandardInvoiceLinesFromAPI(ctx, invoice, request.Input.Lines, resolvedTaxConfigs)
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
			h.logInvoiceEditRejected(ctx, invoice.Type(), request, err)
			return err
		}

		resp, err = h.MapInvoiceToAPI(ctx, billing.NewInvoice(result))

		return err
	}); err != nil {
		return UpdateInvoiceResponse{}, err
	}

	return resp, nil
}

func (h *handler) MapInvoiceToAPI(ctx context.Context, invoice billing.Invoice) (apiv1models.Invoice, error) {
	genericInvoice, err := invoice.AsGenericInvoice()
	if err != nil {
		return apiv1models.Invoice{}, fmt.Errorf("converting invoice to generic invoice: %w", err)
	}

	mapped, err := h.mapInvoicesToAPI(ctx, genericInvoice.GetInvoiceID().Namespace, invoice)
	if err != nil {
		return apiv1models.Invoice{}, err
	}

	return mapped[0], nil
}

// invoiceToMap is an invoice prepared for mapping: standard invoices show their own workflow,
// gathering invoices their customer's current billing profile.
type invoiceToMap struct {
	standard      *billing.StandardInvoice
	gathering     *billing.GatheringInvoice
	mergedProfile billing.CustomerBillingProfile
}

// mapInvoicesToAPI maps invoices with one Tax Code lookup for the default and line tax configs
// of the whole page.
func (h *handler) mapInvoicesToAPI(ctx context.Context, namespace string, invoices ...billing.Invoice) ([]apiv1models.Invoice, error) {
	prepared := make([]invoiceToMap, 0, len(invoices))

	workflows := make([]billing.WorkflowConfig, 0, len(invoices))
	standardLines := make([]*billing.StandardLine, 0, len(invoices))
	gatheringLines := make([]billing.GatheringLine, 0, len(invoices))

	for _, invoice := range invoices {
		switch invoice.Type() {
		case billing.InvoiceTypeStandard:
			standardInvoice, err := invoice.AsStandardInvoice()
			if err != nil {
				return nil, fmt.Errorf("converting invoice to standard invoice: %w", err)
			}

			prepared = append(prepared, invoiceToMap{standard: &standardInvoice})
			workflows = append(workflows, standardInvoice.Workflow.Config)
			standardLines = append(standardLines, standardInvoice.Lines.OrEmpty()...)
		case billing.InvoiceTypeGathering:
			gatheringInvoice, err := invoice.AsGatheringInvoice()
			if err != nil {
				return nil, fmt.Errorf("converting invoice to gathering invoice: %w", err)
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
				return nil, fmt.Errorf("failed to get customer override: %w", err)
			}

			prepared = append(prepared, invoiceToMap{gathering: &gatheringInvoice, mergedProfile: mergedProfile})
			workflows = append(workflows, mergedProfile.EffectiveProfile.WorkflowConfig)
			gatheringLines = append(gatheringLines, gatheringInvoice.Lines.OrEmpty()...)
		default:
			return nil, fmt.Errorf("invalid invoice type: %s", invoice.Type())
		}
	}

	taxCodes, err := billingconvert.ResolveInvoiceTaxCodes(ctx, h.taxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:      namespace,
		Workflows:      workflows,
		StandardLines:  standardLines,
		GatheringLines: gatheringLines,
	})
	if err != nil {
		return nil, err
	}

	out := make([]apiv1models.Invoice, 0, len(prepared))

	for _, i := range prepared {
		var (
			mapped apiv1models.Invoice
			err    error
		)

		if i.standard != nil {
			mapped, err = MapStandardInvoiceToAPI(*i.standard, taxCodes)
		} else {
			mapped, err = MapGatheringInvoiceToAPI(*i.gathering, i.mergedProfile.Customer, i.mergedProfile.EffectiveProfile, taxCodes)
		}

		if err != nil {
			return nil, err
		}

		out = append(out, mapped)
	}

	return out, nil
}

func (h *handler) taxCodesForStandardInvoices(ctx context.Context, namespace string, invoices ...billing.StandardInvoice) (map[string]*taxcode.TaxCode, error) {
	workflows := make([]billing.WorkflowConfig, 0, len(invoices))
	lines := make([]*billing.StandardLine, 0, len(invoices))

	for _, invoice := range invoices {
		workflows = append(workflows, invoice.Workflow.Config)
		lines = append(lines, invoice.Lines.OrEmpty()...)
	}

	return billingconvert.ResolveInvoiceTaxCodes(ctx, h.taxCodeResolver, billingconvert.ResolveInvoiceTaxCodesInput{
		Namespace:     namespace,
		Workflows:     workflows,
		StandardLines: lines,
	})
}

func (h *handler) mapStandardInvoiceWithTaxCodesToAPI(ctx context.Context, invoice billing.StandardInvoice) (apiv1models.Invoice, error) {
	taxCodes, err := h.taxCodesForStandardInvoices(ctx, invoice.Namespace, invoice)
	if err != nil {
		return apiv1models.Invoice{}, err
	}

	return MapStandardInvoiceToAPI(invoice, taxCodes)
}

// MapStandardInvoiceToAPI maps a standard invoice. taxCodes must hold the invoice's default Tax
// Code and every Tax Code its lines' tax configs reference (see ResolveInvoiceTaxCodes) for
// their stripe.code to show.
func MapStandardInvoiceToAPI(invoice billing.StandardInvoice, taxCodes map[string]*taxcode.TaxCode) (apiv1models.Invoice, error) {
	return billingconvert.ToAPIStandardInvoice(invoice, taxCodes)
}

// MapEventInvoiceToAPI maps an invoice event to the v1 invoice shape notifications carry. The event
// has no Stripe code for the invoice default; taxCodes (see ResolveInvoiceTaxCodes) supplies it
// and the lines' stripe.code.
func MapEventInvoiceToAPI(event billing.EventStandardInvoice, taxCodes map[string]*taxcode.TaxCode) (apiv1models.Invoice, error) {
	return billingconvert.ToAPIEventInvoice(event, taxCodes)
}

func mapInvoiceExpandsToEntity(expand []apiv1models.InvoiceExpand) billing.InvoiceExpands {
	if len(expand) == 0 {
		return nil
	}

	return billing.InvoiceExpands{}.
		SetOrUnsetIf(slices.Contains(expand, apiv1models.InvoiceExpandLines), billing.InvoiceExpandLines)
}

func mergeInvoiceSupplierFromAPI(existing billing.SupplierContact, updatedSupplier apiv1models.BillingPartyReplaceUpdate) billing.SupplierContact {
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

func mergeInvoiceCustomerFromAPI(existing billing.InvoiceCustomer, updatedCustomer apiv1models.BillingPartyReplaceUpdate) billing.InvoiceCustomer {
	existing.Name = lo.FromPtr(updatedCustomer.Name)

	if updatedCustomer.Addresses == nil || len(*updatedCustomer.Addresses) == 0 {
		existing.BillingAddress = nil
	} else {
		mappedAddress := customerhttpdriver.MapAddress(&(*updatedCustomer.Addresses)[0])
		existing.BillingAddress = mappedAddress
	}

	return existing
}

func mergeInvoiceWorkflowFromAPI(existing billing.InvoiceWorkflow, updatedWorkflow apiv1models.InvoiceWorkflowReplaceUpdate) (billing.InvoiceWorkflow, error) {
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
