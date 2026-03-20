package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) GetByMetas(ctx context.Context, input usagebased.GetByMetasInput) ([]usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.Charge, error) {
		return s.adapter.GetByMetas(ctx, input)
	})
}
