package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.InvoiceItemService = (*Service)(nil)

func (s *Service) CreateInvoiceItems(ctx context.Context, input billing.CreateInvoiceItemsInput) ([]billing.InvoiceItem, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return billing.WithTx(ctx, s.adapter, func(ctx context.Context, adapter billing.TxAdapter) ([]billing.InvoiceItem, error) {
		return adapter.CreateInvoiceItems(ctx, input)
	})
}
