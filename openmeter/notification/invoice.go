package notification

import (
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	EventTypeInvoiceCreated EventType = "invoice.created"
	EventTypeInvoiceUpdated EventType = "invoice.updated"
)

type InvoicePayload = billing.EventStandardInvoice

var (
	_ models.Validator                          = (*InvoiceRuleConfig)(nil)
	_ models.CustomValidator[InvoiceRuleConfig] = (*InvoiceRuleConfig)(nil)
)

type InvoiceRuleConfig struct{}

func (c InvoiceRuleConfig) ValidateWith(validators ...models.ValidatorFunc[InvoiceRuleConfig]) error {
	return models.Validate(c, validators...)
}

func (c InvoiceRuleConfig) Validate() error {
	return nil
}
