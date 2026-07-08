package meta

import "github.com/qmuntal/stateless"

type Trigger = stateless.Trigger

var (
	TriggerNext                   Trigger = "next"
	TriggerInvoiceCreated         Trigger = "invoice_created"
	TriggerCollectionCompleted    Trigger = "collection_completed"
	TriggerInvoiceIssued          Trigger = "invoice_issued"
	TriggerAllPaymentsSettled     Trigger = "all_payments_settled"
	TriggerLineManualEdit         Trigger = "line_manual_edit"
	TriggerShrinkToRealizedPeriod Trigger = "shrink_to_realized_period"
	TriggerAttachInvoiceLine      Trigger = "attach_invoice_line"
)
