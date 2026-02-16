package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) UpsertChargesByChildUniqueReferenceID(ctx context.Context, input charges.UpsertChargesByChildUniqueReferenceIDInput) (charges.Charges, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		return s.adapter.UpsertChargesByChildUniqueReferenceID(ctx, input)
	})
}
