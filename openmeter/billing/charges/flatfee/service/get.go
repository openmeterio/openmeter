package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) GetByMetas(ctx context.Context, input flatfee.GetByMetasInput) ([]flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]flatfee.Charge, error) {
		return s.adapter.GetByMetas(ctx, input)
	})
}
