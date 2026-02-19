package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) GetChargeByID(ctx context.Context, input charges.ChargeID) (charges.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charge, error) {
		return s.adapter.GetChargeByID(ctx, input)
	})
}

func (s *service) GetChargesByIDs(ctx context.Context, ns string, ids []string) (charges.Charges, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		return s.adapter.GetChargesByIDs(ctx, ns, ids)
	})
}
