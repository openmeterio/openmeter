package notification

import (
	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	EventTypeInvoiceCreated EventType = "invoice.created"
	EventTypeInvoiceUpdated EventType = "invoice.updated"
)

var (
	_ models.Validator                       = (*InvoicePayload)(nil)
	_ models.CustomValidator[InvoicePayload] = (*InvoicePayload)(nil)
)

type InvoicePayload struct {
	apiv1models.Invoice
}

func (ip InvoicePayload) ValidateWith(validators ...models.ValidatorFunc[InvoicePayload]) error {
	return models.Validate(ip, validators...)
}

func (ip InvoicePayload) Validate() error {
	return nil
}

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
