package meta

import "github.com/qmuntal/stateless"

type Trigger = stateless.Trigger

var (
	TriggerNext                  Trigger = "next"
	TriggerPartialInvoiceCreated Trigger = "partial_invoice_created"
	TriggerFinalInvoiceCreated   Trigger = "final_invoice_created"
	TriggerCollectionCompleted   Trigger = "collection_completed"
	TriggerInvoiceIssued         Trigger = "invoice_issued"
	TriggerAllPaymentsSettled    Trigger = "all_payments_settled"
)
