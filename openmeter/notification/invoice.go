package notification

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

const (
	EventTypeInvoiceCreated EventType = "invoice.created"
	EventTypeInvoiceUpdated EventType = "invoice.updated"
)

type InvoicePayload = billing.EventInvoice

type InvoiceRuleConfig struct{}

func (c InvoiceRuleConfig) Validate(ctx context.Context, service Service, namespace string) error {
	return nil
}
