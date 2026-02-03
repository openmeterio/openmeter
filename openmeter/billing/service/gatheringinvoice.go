package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ billing.GatheringInvoiceService = (*Service)(nil)

func (s *Service) ListGatheringInvoices(ctx context.Context, input billing.ListGatheringInvoicesInput) (pagination.Result[billing.GatheringInvoice], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[billing.GatheringInvoice]{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (pagination.Result[billing.GatheringInvoice], error) {
		return s.adapter.ListGatheringInvoices(ctx, input)
	})
}
