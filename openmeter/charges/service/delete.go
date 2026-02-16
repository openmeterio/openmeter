package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) DeleteChargesByUniqueReferenceID(ctx context.Context, input charges.DeleteChargesByUniqueReferenceIDInput) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.DeleteChargesByUniqueReferenceID(ctx, input)
	})
}
