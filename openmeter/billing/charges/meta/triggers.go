package meta

import "github.com/qmuntal/stateless"

type Trigger = stateless.Trigger

var (
	TriggerNext                   Trigger = "next"
	TriggerInvoiceCreated         Trigger = "invoice_created"
	TriggerCollectionCompleted    Trigger = "collection_completed"
	TriggerInvoiceFinalizing      Trigger = "invoice_finalizing"
	TriggerInvoiceIssued          Trigger = "invoice_issued"
	TriggerLineManualEdit         Trigger = "line_manual_edit"
	TriggerSetOverride            Trigger = "set_override"
	TriggerClearOverride          Trigger = "clear_override"
	TriggerShrinkToRealizedPeriod Trigger = "shrink_to_realized_period"
	TriggerAttachInvoiceLine      Trigger = "attach_invoice_line"
)
