package billing

import (
	"fmt"
	"slices"

	"github.com/qmuntal/stateless"
)

type InvoiceTrigger = stateless.Trigger

var (
	// TriggerRetry is used to retry a state transition that failed, used by the end user to invoke it manually
	TriggerRetry InvoiceTrigger = "trigger_retry"
	// TriggerApprove is used to approve a state manually
	TriggerApprove InvoiceTrigger = "trigger_approve"
	// TriggerNext is used to advance the invoice to the next state if automatically possible
	TriggerNext InvoiceTrigger = "trigger_next"
	// TriggerFailed is used to trigger the failure state transition associated with the current state
	TriggerFailed InvoiceTrigger = "trigger_failed"
	// TriggerUpdated is used to trigger a change in the invoice (we are using this to calculate the immutable states
	// and trigger re-validation)
	TriggerUpdated InvoiceTrigger = "trigger_updated"
	// TriggerSnapshotQuantities is used to snapshot the quantities for usage based line items
	TriggerSnapshotQuantities InvoiceTrigger = "trigger_snapshot_quantities"
	// triggerDelete is used to delete the invoice
	TriggerDelete InvoiceTrigger = "trigger_delete"

	// TODO[OM-989]: we should have a triggerAsyncNext to signify that a transition should be done asynchronously (
	// e.g. the invoice needs to be synced to an external system such as stripe)

	// TriggerPaid is used to signify that the invoice has been paid
	TriggerPaid InvoiceTrigger = "trigger_paid"
	// TriggerActionRequired is used to signify that the invoice requires action
	TriggerActionRequired InvoiceTrigger = "trigger_action_required"

	// TriggerPaymentUncollectible is used to signify that the invoice is uncollectible
	TriggerPaymentUncollectible InvoiceTrigger = "trigger_payment_uncollectible"
	// TriggerPaymentOverdue is used to signify that the invoice is overdue
	TriggerPaymentOverdue InvoiceTrigger = "trigger_payment_overdue"

	// TriggerVoid is used to signify that the invoice has been voided (e.g. created by mistake)
	TriggerVoid InvoiceTrigger = "trigger_void"
)

type StandardInvoiceOperation string

const (
	StandardInvoiceOpValidate        StandardInvoiceOperation = "validate"
	StandardInvoiceOpSync            StandardInvoiceOperation = "sync"
	StandardInvoiceOpDelete          StandardInvoiceOperation = "delete"
	StandardInvoiceOpFinalize        StandardInvoiceOperation = "finalize"
	StandardInvoiceOpInitiatePayment StandardInvoiceOperation = "initiate_payment"

	StandardInvoiceOpPostAdvanceHook StandardInvoiceOperation = "post_advance_hook"
	StandardInvoiceOpTriggerInvoice  StandardInvoiceOperation = "trigger_invoice"
)

var StandardInvoiceOperations = []StandardInvoiceOperation{
	StandardInvoiceOpValidate,
	StandardInvoiceOpSync,
	StandardInvoiceOpDelete,
	StandardInvoiceOpFinalize,
	StandardInvoiceOpInitiatePayment,

	StandardInvoiceOpPostAdvanceHook,
	StandardInvoiceOpTriggerInvoice,
}

func (o StandardInvoiceOperation) Validate() error {
	if !slices.Contains(StandardInvoiceOperations, o) {
		return fmt.Errorf("invalid standard invoice operation: %s", o)
	}

	return nil
}
