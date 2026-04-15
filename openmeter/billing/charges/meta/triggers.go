package meta

import "github.com/qmuntal/stateless"

type Trigger = stateless.Trigger

var (
	TriggerNext                Trigger = "next"
	TriggerInvoiceCreated      Trigger = "invoice_created"
	TriggerCollectionCompleted Trigger = "collection_completed"
	TriggerInvoiceIssued       Trigger = "invoice_issued"
)
