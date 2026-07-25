package billingworkercollect

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func (a *InvoiceCollector) HandleCollectCustomerInvoicesEvent(ctx context.Context, event *billing.CollectCustomerInvoicesEvent) error {
	if event == nil {
		return nil
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid collect customer invoices event: %w", err)
	}

	_, err := a.CollectCustomerInvoice(ctx, CollectCustomerInvoiceInput{
		CustomerID: customer.CustomerID{
			Namespace: event.Namespace,
			ID:        event.CustomerID,
		},
		AsOf: event.AsOf,
	})

	return err
}
