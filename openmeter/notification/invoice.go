package notification

import "github.com/openmeterio/openmeter/openmeter/billing"

const (
	EventTypeInvoiceCreated EventType = "invoice.created"
	EventTypeInvoiceUpdated EventType = "invoice.updated"
)

type InvoicePayload = billing.EventInvoice

type InvoiceRuleConfig struct{}
