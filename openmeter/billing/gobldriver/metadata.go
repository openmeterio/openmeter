package gobldriver

import "github.com/invopop/gobl/cbc"

const (
	// Invoice constants
	InvoiceIDKey cbc.Key = "openmeter-invoice-id"

	// InvoiceItem note constants
	InvoiceItemNoteSourceOpenmeter cbc.Key = "openmeter-billing"

	// Lifecycle note constants

	// InvoiceItemCodeLifecycle is the code for the lifecycle of an invoice item (e.g. billing period, created, etc.)
	InvoiceItemCodeLifecycle string = "lifecycle"

	// All metadata field names must match ^(?:[a-z]|[a-z0-9][a-z0-9-+]*[a-z0-9])$`
	// see: https://github.com/invopop/gobl/blob/d10f919fd2d9b59972aebb2295c25030eb1ba38e/cbc/key.go#L20
	InvoiceItemBillingPeriodStart cbc.Key = "billing-period-start"
	InvoiceItemBillingPeriodEnd   cbc.Key = "billing-period-end"
	InvoiceItemInvoiceAt          cbc.Key = "invoice-at"
	InvoiceItemCreated            cbc.Key = "created-at"
	InvoiceItemUpdated            cbc.Key = "updated-at"

	// Entity note constants

	// InvoiceItemCodeEntity is the code for the entity of an invoice item (e.g. id, etc.)
	InvoiceItemCodeEntity string = "entity"

	InvoiceItemEntityID cbc.Key = "entity-id"
)
