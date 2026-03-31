package invoicesync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// Executor sequentially executes sync plan operations against the Stripe API.
type Executor struct {
	Adapter Adapter
	Logger  *slog.Logger
}

// ExecuteNextOperationResult holds the result of executing the next operation.
type ExecuteNextOperationResult struct {
	Done      bool
	PlanID    string
	Failed    bool
	FailError string

	// ExternalIDs populated by InvoiceCreate and LineItemAdd operations for incremental sync-back.
	InvoicingExternalID     *string
	LineExternalIDs         map[string]string
	LineDiscountExternalIDs map[string]string
}

// ExecuteNextOperation finds the next pending operation in the plan and executes it.
// Returns done=true when all operations are completed or the plan has failed.
func (e *Executor) ExecuteNextOperation(ctx context.Context, stripeClient stripeclient.StripeAppClient, plan *SyncPlan) (*ExecuteNextOperationResult, error) {
	// Update plan status to executing if still pending
	if plan.Status == PlanStatusPending {
		if err := e.Adapter.UpdatePlanStatus(ctx, plan.ID, PlanStatusExecuting, nil); err != nil {
			return nil, fmt.Errorf("updating plan status to executing: %w", err)
		}
	}

	op, err := e.Adapter.GetNextPendingOperation(ctx, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("getting next pending operation: %w", err)
	}

	if op == nil {
		// All operations completed
		if err := e.Adapter.CompletePlan(ctx, plan.ID); err != nil {
			return nil, fmt.Errorf("completing plan: %w", err)
		}
		return &ExecuteNextOperationResult{Done: true, PlanID: plan.ID}, nil
	}

	e.Logger.InfoContext(ctx, "executing sync operation",
		"plan_id", plan.ID,
		"op_id", op.ID,
		"op_type", op.Type,
		"sequence", op.Sequence,
	)

	// Resolve the Stripe invoice ID from prior operations if needed
	stripeInvoiceID, err := e.resolveStripeInvoiceID(ctx, plan, op)
	if err != nil {
		return nil, fmt.Errorf("resolving stripe invoice ID: %w", err)
	}

	response, execErr := e.executeOperation(ctx, stripeClient, op, stripeInvoiceID)

	if execErr != nil {
		if isRetryableError(execErr) {
			return nil, fmt.Errorf("retryable error executing operation %s: %w", op.Type, execErr)
		}

		// Non-retryable error: fail the operation and plan.
		// If the DB updates fail, return the error so the transaction rolls back and Kafka retries.
		errMsg := execErr.Error()
		if failErr := e.Adapter.FailOperation(ctx, op.ID, errMsg); failErr != nil {
			return nil, fmt.Errorf("marking operation as failed: %w (original: %s)", failErr, errMsg)
		}
		if failErr := e.Adapter.FailPlan(ctx, plan.ID, errMsg); failErr != nil {
			return nil, fmt.Errorf("marking plan as failed: %w (original: %s)", failErr, errMsg)
		}
		return &ExecuteNextOperationResult{
			Done:      true,
			PlanID:    plan.ID,
			Failed:    true,
			FailError: errMsg,
		}, nil
	}

	// Mark operation as completed
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("marshaling response: %w", err)
	}
	if err := e.Adapter.CompleteOperation(ctx, op.ID, responseBytes); err != nil {
		return nil, fmt.Errorf("completing operation: %w", err)
	}

	// Extract external IDs from the response for incremental sync-back
	result := &ExecuteNextOperationResult{Done: false, PlanID: plan.ID}
	switch resp := response.(type) {
	case *InvoiceCreateResponse:
		result.InvoicingExternalID = &resp.StripeInvoiceID
	case *LineItemAddResponse:
		result.LineExternalIDs = resp.LineExternalIDs
		result.LineDiscountExternalIDs = resp.LineDiscountExternalIDs
	}

	return result, nil
}

// payloadWithStripeInvoiceID is used to extract the StripeInvoiceID from any operation payload
// without needing to know the concrete type. All payload types share this JSON field.
type payloadWithStripeInvoiceID struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// resolveStripeInvoiceID resolves the Stripe invoice ID for operations that depend on a prior InvoiceCreate.
func (e *Executor) resolveStripeInvoiceID(ctx context.Context, plan *SyncPlan, op *SyncOperation) (string, error) {
	if op.Type == OpTypeInvoiceCreate {
		return "", nil
	}

	// All non-create payloads share a stripe_invoice_id field — try extracting it
	var p payloadWithStripeInvoiceID
	if err := json.Unmarshal(op.Payload, &p); err == nil && p.StripeInvoiceID != "" {
		return p.StripeInvoiceID, nil
	}

	// Fall back: look for a completed InvoiceCreate in this plan's operations.
	// NOTE: plan.Operations reflects the state at fetch time. The handler processes one
	// operation per event and re-fetches the plan each time, so this data is fresh.
	// If the executor is ever changed to process multiple ops in a loop without re-fetching,
	// this fallback would need to query the DB instead.
	for _, planOp := range plan.Operations {
		if planOp.Type == OpTypeInvoiceCreate && planOp.Status == OpStatusCompleted && planOp.StripeResponse != nil {
			var resp InvoiceCreateResponse
			if err := json.Unmarshal(planOp.StripeResponse, &resp); err == nil && resp.StripeInvoiceID != "" {
				return resp.StripeInvoiceID, nil
			}
		}
	}

	return "", fmt.Errorf("could not resolve Stripe invoice ID for operation %s (sequence %d)", op.Type, op.Sequence)
}

// resolveInvoiceID picks the Stripe invoice ID from the payload (if present) or falls back to the resolved one.
func resolveInvoiceID(payload json.RawMessage, fallback string) string {
	var p payloadWithStripeInvoiceID
	if err := json.Unmarshal(payload, &p); err == nil && p.StripeInvoiceID != "" {
		return p.StripeInvoiceID
	}
	return fallback
}

func (e *Executor) executeOperation(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (any, error) {
	switch op.Type {
	case OpTypeInvoiceCreate:
		return e.executeInvoiceCreate(ctx, stripeClient, op)
	case OpTypeInvoiceUpdate:
		return e.executeInvoiceUpdate(ctx, stripeClient, op, stripeInvoiceID)
	case OpTypeInvoiceDelete:
		return e.executeInvoiceDelete(ctx, stripeClient, op, stripeInvoiceID)
	case OpTypeInvoiceFinalize:
		return e.executeInvoiceFinalize(ctx, stripeClient, op, stripeInvoiceID)
	case OpTypeLineItemAdd:
		return e.executeLineItemAdd(ctx, stripeClient, op, stripeInvoiceID)
	case OpTypeLineItemUpdate:
		return e.executeLineItemUpdate(ctx, stripeClient, op, stripeInvoiceID)
	case OpTypeLineItemRemove:
		return e.executeLineItemRemove(ctx, stripeClient, op, stripeInvoiceID)
	default:
		return nil, fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

func (e *Executor) executeInvoiceCreate(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation) (*InvoiceCreateResponse, error) {
	var payload InvoiceCreatePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling create payload: %w", err)
	}

	var defaultPaymentMethod *string
	if payload.StripeDefaultPaymentMethodID != "" {
		defaultPaymentMethod = &payload.StripeDefaultPaymentMethodID
	}

	invoice, err := stripeClient.CreateInvoice(ctx, stripeclient.CreateInvoiceInput{
		AppID: app.AppID{
			Namespace: payload.Namespace,
			ID:        payload.AppID,
		},
		CustomerID: customer.CustomerID{
			Namespace: payload.Namespace,
			ID:        payload.CustomerID,
		},
		InvoiceID:                    payload.InvoiceID,
		AutomaticTaxEnabled:          payload.AutomaticTaxEnabled,
		CollectionMethod:             payload.CollectionMethod,
		Currency:                     currencyx.Code(payload.Currency),
		DaysUntilDue:                 payload.DaysUntilDue,
		StripeCustomerID:             payload.StripeCustomerID,
		StripeDefaultPaymentMethodID: defaultPaymentMethod,
	})
	if err != nil {
		return nil, fmt.Errorf("creating invoice in stripe: %w", err)
	}

	return &InvoiceCreateResponse{
		StripeInvoiceID: invoice.ID,
		InvoiceNumber:   invoice.Number,
	}, nil
}

func (e *Executor) executeInvoiceUpdate(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (*InvoiceUpdateResponse, error) {
	var payload InvoiceUpdatePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling update payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	invoice, err := stripeClient.UpdateInvoice(ctx, stripeclient.UpdateInvoiceInput{
		AutomaticTaxEnabled: payload.AutomaticTaxEnabled,
		StripeInvoiceID:     invoiceID,
	})
	if err != nil {
		return nil, fmt.Errorf("updating invoice in stripe: %w", err)
	}

	return &InvoiceUpdateResponse{
		StripeInvoiceID: invoice.ID,
		InvoiceNumber:   invoice.Number,
	}, nil
}

func (e *Executor) executeInvoiceDelete(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (any, error) {
	var payload InvoiceDeletePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling delete payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	err := stripeClient.DeleteInvoice(ctx, stripeclient.DeleteInvoiceInput{
		StripeInvoiceID: invoiceID,
	})
	if err != nil {
		return nil, fmt.Errorf("deleting invoice in stripe: %w", err)
	}

	return &InvoiceDeleteResponse{StripeInvoiceID: invoiceID}, nil
}

func (e *Executor) executeInvoiceFinalize(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (*InvoiceFinalizeResponse, error) {
	var payload InvoiceFinalizePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling finalize payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	invoice, err := stripeClient.FinalizeInvoice(ctx, stripeclient.FinalizeInvoiceInput{
		StripeInvoiceID: invoiceID,
		AutoAdvance:     payload.AutoAdvance,
	})
	if err != nil {
		// Handle tax location error: disable tax and retry
		if stripeclient.IsStripeInvoiceCustomerTaxLocationInvalidError(err) {
			if payload.TaxEnforced {
				return nil, fmt.Errorf("tax enforced but stripe tax returns error: %w", err)
			}

			// Disable tax and retry
			_, updateErr := stripeClient.UpdateInvoice(ctx, stripeclient.UpdateInvoiceInput{
				AutomaticTaxEnabled: false,
				StripeInvoiceID:     invoiceID,
			})
			if updateErr != nil {
				return nil, fmt.Errorf("disabling tax for invoice: %w", updateErr)
			}

			invoice, err = stripeClient.FinalizeInvoice(ctx, stripeclient.FinalizeInvoiceInput{
				StripeInvoiceID: invoiceID,
				AutoAdvance:     payload.AutoAdvance,
			})
			if err != nil {
				return nil, fmt.Errorf("finalizing invoice after disabling tax: %w", err)
			}
		} else {
			return nil, fmt.Errorf("finalizing invoice in stripe: %w", err)
		}
	}

	result := &InvoiceFinalizeResponse{
		InvoiceNumber: invoice.Number,
	}
	if invoice.PaymentIntent != nil {
		result.PaymentExternalID = &invoice.PaymentIntent.ID
	}

	return result, nil
}

func (e *Executor) executeLineItemAdd(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (*LineItemAddResponse, error) {
	var payload LineItemAddPayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling line add payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	// Convert our serializable params to Stripe params
	stripeParams := make([]*stripe.InvoiceItemParams, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		params := toStripeInvoiceItemParams(line.Description, line.Amount, line.Currency,
			line.PeriodStart, line.PeriodEnd, line.Metadata, line.TaxBehavior, line.TaxCode)
		params.Customer = stripe.String(line.CustomerID)
		stripeParams = append(stripeParams, params)
	}

	newLines, err := stripeClient.AddInvoiceLines(ctx, stripeclient.AddInvoiceLinesInput{
		StripeInvoiceID: invoiceID,
		Lines:           stripeParams,
	})
	if err != nil {
		return nil, fmt.Errorf("adding line items to stripe: %w", err)
	}

	result := &LineItemAddResponse{
		LineExternalIDs:         make(map[string]string, len(newLines)),
		LineDiscountExternalIDs: make(map[string]string),
	}

	for _, sl := range newLines {
		lineID, ok := sl.Metadata[LineMetadataID]
		if !ok {
			continue
		}
		lineType := sl.Metadata[LineMetadataType]
		if lineType == LineMetadataTypeDiscount {
			result.LineDiscountExternalIDs[lineID] = sl.LineID
		} else {
			result.LineExternalIDs[lineID] = sl.LineID
		}
	}

	return result, nil
}

func (e *Executor) executeLineItemUpdate(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (any, error) {
	var payload LineItemUpdatePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling line update payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	stripeParams := make([]*stripeclient.StripeInvoiceItemWithID, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		stripeParams = append(stripeParams, &stripeclient.StripeInvoiceItemWithID{
			ID: line.ID,
			InvoiceItemParams: toStripeInvoiceItemParams(line.Description, line.Amount, line.Currency,
				line.PeriodStart, line.PeriodEnd, line.Metadata, line.TaxBehavior, line.TaxCode),
		})
	}

	_, err := stripeClient.UpdateInvoiceLines(ctx, stripeclient.UpdateInvoiceLinesInput{
		StripeInvoiceID: invoiceID,
		Lines:           stripeParams,
	})
	if err != nil {
		return nil, fmt.Errorf("updating line items in stripe: %w", err)
	}

	return &LineItemUpdateResponse{StripeInvoiceID: invoiceID}, nil
}

func (e *Executor) executeLineItemRemove(ctx context.Context, stripeClient stripeclient.StripeAppClient, op *SyncOperation, stripeInvoiceID string) (any, error) {
	var payload LineItemRemovePayload
	if err := json.Unmarshal(op.Payload, &payload); err != nil {
		return nil, fmt.Errorf("unmarshaling line remove payload: %w", err)
	}

	invoiceID := resolveInvoiceID(op.Payload, stripeInvoiceID)

	err := stripeClient.RemoveInvoiceLines(ctx, stripeclient.RemoveInvoiceLinesInput{
		StripeInvoiceID: invoiceID,
		Lines:           payload.LineIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("removing line items from stripe: %w", err)
	}

	return &LineItemRemoveResponse{StripeInvoiceID: invoiceID}, nil
}

// toStripeInvoiceItemParams builds a Stripe InvoiceItemParams from common line fields.
func toStripeInvoiceItemParams(description string, amount int64, currency string, periodStart, periodEnd int64, metadata map[string]string, taxBehavior, taxCode *string) *stripe.InvoiceItemParams {
	params := &stripe.InvoiceItemParams{
		Description: lo.ToPtr(description),
		Amount:      lo.ToPtr(amount),
		Currency:    lo.ToPtr(currency),
		Period: &stripe.InvoiceItemPeriodParams{
			Start: lo.ToPtr(periodStart),
			End:   lo.ToPtr(periodEnd),
		},
		Metadata: metadata,
	}
	if taxBehavior != nil {
		params.TaxBehavior = taxBehavior
	}
	if taxCode != nil {
		params.TaxCode = taxCode
	}
	return params
}

// isRetryableError checks if a Stripe error is retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var stripeErr *stripe.Error
	if !errors.As(err, &stripeErr) {
		// Non-Stripe errors (validation, marshaling, business logic) are not retryable.
		// Genuine network errors from the Stripe SDK are wrapped as *stripe.Error with 5xx status.
		return false
	}

	switch stripeErr.HTTPStatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}

	return false
}

// BuildUpsertResultFromPlan constructs a UpsertStandardInvoiceResult from a completed sync plan.
func BuildUpsertResultFromPlan(plan *SyncPlan) (*billing.UpsertStandardInvoiceResult, error) {
	result := billing.NewUpsertStandardInvoiceResult()

	for _, op := range plan.Operations {
		if op.Status != OpStatusCompleted || op.StripeResponse == nil {
			continue
		}

		switch op.Type {
		case OpTypeInvoiceCreate:
			var resp InvoiceCreateResponse
			if err := json.Unmarshal(op.StripeResponse, &resp); err != nil {
				return nil, fmt.Errorf("unmarshaling create response: %w", err)
			}
			result.SetExternalID(resp.StripeInvoiceID)
			result.SetInvoiceNumber(resp.InvoiceNumber)

		case OpTypeInvoiceUpdate:
			var resp InvoiceUpdateResponse
			if err := json.Unmarshal(op.StripeResponse, &resp); err != nil {
				return nil, fmt.Errorf("unmarshaling update response: %w", err)
			}
			result.SetExternalID(resp.StripeInvoiceID)
			result.SetInvoiceNumber(resp.InvoiceNumber)

		case OpTypeLineItemAdd:
			var resp LineItemAddResponse
			if err := json.Unmarshal(op.StripeResponse, &resp); err != nil {
				return nil, fmt.Errorf("unmarshaling line add response: %w", err)
			}
			for lineID, externalID := range resp.LineExternalIDs {
				result.AddLineExternalID(lineID, externalID)
			}
			for discountID, externalID := range resp.LineDiscountExternalIDs {
				result.AddLineDiscountExternalID(discountID, externalID)
			}
		}
	}

	return result, nil
}

// BuildFinalizeResultFromPlan constructs a FinalizeStandardInvoiceResult from a completed issuing sync plan.
func BuildFinalizeResultFromPlan(plan *SyncPlan) (*billing.FinalizeStandardInvoiceResult, error) {
	result := billing.NewFinalizeStandardInvoiceResult()

	for _, op := range plan.Operations {
		if op.Status != OpStatusCompleted || op.StripeResponse == nil {
			continue
		}

		switch op.Type {
		case OpTypeInvoiceFinalize:
			var resp InvoiceFinalizeResponse
			if err := json.Unmarshal(op.StripeResponse, &resp); err != nil {
				return nil, fmt.Errorf("unmarshaling finalize response: %w", err)
			}
			result.SetInvoiceNumber(resp.InvoiceNumber)
			if resp.PaymentExternalID != nil {
				result.SetPaymentExternalID(*resp.PaymentExternalID)
			}

		case OpTypeInvoiceUpdate:
			var resp InvoiceUpdateResponse
			if err := json.Unmarshal(op.StripeResponse, &resp); err != nil {
				return nil, fmt.Errorf("unmarshaling update response: %w", err)
			}
			result.SetInvoiceNumber(resp.InvoiceNumber)
		}
	}

	return result, nil
}
