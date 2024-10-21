package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.InvoiceItemService = (*Service)(nil)

func (s *Service) CreateInvoiceItems(ctx context.Context, input billing.CreateInvoiceItemsInput) ([]billingentity.InvoiceItem, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{
			Err: err,
		}
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]billingentity.InvoiceItem, error) {
		return s.adapter.CreateInvoiceItems(ctx, input)
	})
}
