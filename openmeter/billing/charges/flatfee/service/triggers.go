package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) AdvanceCharge(ctx context.Context, input flatfee.AdvanceChargeInput) (*flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*flatfee.Charge, error) {
		key, err := charges.NewLockKeyForCharge(input.ChargeID)
		if err != nil {
			return nil, fmt.Errorf("get charge lock key: %w", err)
		}

		if err := s.locker.LockForTX(ctx, key); err != nil {
			return nil, fmt.Errorf("lock charge: %w", err)
		}

		fetchedCharges, err := s.adapter.GetByIDs(ctx, flatfee.GetByIDsInput{
			Namespace: input.ChargeID.Namespace,
			IDs:       []string{input.ChargeID.ID},
			Expands:   meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("get charge: %w", err)
		}

		if len(fetchedCharges) == 0 {
			return nil, fmt.Errorf("charge not found [id=%s]", input.ChargeID.ID)
		}

		charge := fetchedCharges[0]

		if charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
			return nil, fmt.Errorf("charge %s is not credit_only (settlement_mode=%s)", charge.ID, charge.Intent.SettlementMode)
		}

		stateMachine, err := NewCreditsOnlyStateMachine(CreditsOnlyStateMachineConfig{
			Charge:  charge,
			Service: s,
		})
		if err != nil {
			return nil, fmt.Errorf("new credits only state machine: %w", err)
		}

		return stateMachine.AdvanceUntilStateStable(ctx)
	})
}
