package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) AdvanceCharge(ctx context.Context, input usagebased.AdvanceChargeInput) (*usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*usagebased.Charge, error) {
		key, err := charges.NewLockKeyForCharge(input.ChargeID)
		if err != nil {
			return nil, fmt.Errorf("get charge lock key: %w", err)
		}

		if err := s.locker.LockForTX(ctx, key); err != nil {
			return nil, fmt.Errorf("lock charge: %w", err)
		}

		charge, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
			ChargeID: input.ChargeID,
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("get charge: %w", err)
		}

		switch charge.Intent.SettlementMode {
		case productcatalog.CreditOnlySettlementMode:
		case productcatalog.CreditThenInvoiceSettlementMode:
			return nil, models.NewGenericNotImplementedError(
				fmt.Errorf("advancing usage based charge with settlement mode %s is not supported [charge_id=%s]", charge.Intent.SettlementMode, charge.ID),
			)
		default:
			return nil, fmt.Errorf("unsupported settlement mode %s [charge_id=%s]", charge.Intent.SettlementMode, charge.ID)
		}

		stateMachine, err := NewCreditsOnlyStateMachine(StateMachineConfig{
			Charge:           charge,
			Service:          s,
			CustomerOverride: input.CustomerOverride,
			FeatureMeter:     input.FeatureMeter,
		})
		if err != nil {
			return nil, fmt.Errorf("new credits only state machine: %w", err)
		}

		return stateMachine.AdvanceUntilStateStable(ctx)
	})
}
