package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) MarkVoided(ctx context.Context, input creditpurchase.MarkVoidedInput) (creditpurchase.ChargeBase, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.ChargeBase{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.ChargeBase, error) {
		charge, err := s.adapter.GetByID(ctx, creditpurchase.GetByIDInput{
			ChargeID: input.ChargeID,
			Expands:  meta.ExpandNone,
		})
		if err != nil {
			return creditpurchase.ChargeBase{}, fmt.Errorf("getting credit purchase charge before voiding [id=%s]: %w", input.ChargeID.ID, err)
		}

		if charge.State.VoidedAt != nil {
			return charge.ChargeBase, nil
		}

		if err := charge.Validate(); err != nil {
			return creditpurchase.ChargeBase{}, fmt.Errorf("validating credit purchase charge before voiding [id=%s]: %w", input.ChargeID.ID, err)
		}

		return s.adapter.MarkVoided(ctx, creditpurchase.MarkVoidedAdapterInput{
			Charge:   charge,
			VoidedAt: input.VoidedAt,
		})
	})
}
