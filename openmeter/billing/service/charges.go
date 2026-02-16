package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.ChargesService = (*Service)(nil)

func (s *Service) SetChargeIDsOnInvoiceLines(ctx context.Context, input billing.SetChargeIDsOnInvoiceLinesInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.SetChargeIDsOnInvoiceLines(ctx, input)
	})
}

func (s *Service) SetChargeIDsOnSplitlineGroups(ctx context.Context, input billing.SetChargeIDsOnSplitlineGroupsInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.SetChargeIDsOnSplitlineGroups(ctx, input)
	})
}
