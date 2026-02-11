package billingservice

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.LockableService = (*Service)(nil)

func (s *Service) WithLock(ctx context.Context, customerID customer.CustomerID, fn func(ctx context.Context) error) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		err := transactionForInvoiceManipulationNoValue(ctx, s, customerID, func(ctx context.Context) error {
			return fn(ctx)
		})
		if err != nil {
			s.logger.WarnContext(ctx, "error while executing locked transaction, rolling back", "error", err)
		}

		return err
	})
}
