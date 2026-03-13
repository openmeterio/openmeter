package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) GetByIDs(ctx context.Context, input creditpurchase.GetByIDsInput) ([]creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]creditpurchase.Charge, error) {
		return s.adapter.GetByIDs(ctx, input)
	})
}
