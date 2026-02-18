package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) CreateCharges(ctx context.Context, input charges.Charges) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		return s.adapter.CreateCharges(ctx, input)
	})
}
