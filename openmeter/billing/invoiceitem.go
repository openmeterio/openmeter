package billing

import (
	"errors"
	"fmt"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type CreateInvoiceItemsInput struct {
	InvoiceID *string
	Namespace string
	Items     []billingentity.InvoiceItem
}

func (c CreateInvoiceItemsInput) Validate() error {
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}

	for idx, item := range c.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item[%d]: %w", idx, err)
		}
	}

	return nil
}
