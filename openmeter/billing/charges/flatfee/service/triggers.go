package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) AdvanceCharge(ctx context.Context, input flatfee.AdvanceChargeInput) (*flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) {
		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge:               charge,
			Adapter:              s.adapter,
			Realizations:         s.realizations,
			Service:              s,
			CreditNotesSupported: s.creditNotesSupported.Load(),
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}

		return stateMachine.AdvanceUntilStateStable(ctx)
	})
}

func (s *service) TriggerPatch(ctx context.Context, chargeID meta.ChargeID, patch meta.Patch) (meta.TriggerPatchResult[flatfee.Charge], error) {
	if err := patch.Validate(); err != nil {
		return meta.TriggerPatchResult[flatfee.Charge]{}, fmt.Errorf("patch: %w", err)
	}

	if err := chargeID.Validate(); err != nil {
		return meta.TriggerPatchResult[flatfee.Charge]{}, fmt.Errorf("chargeID: %w", err)
	}

	var result meta.TriggerPatchResult[flatfee.Charge]

	charge, err := s.withLockedCharge(ctx, chargeID, func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error) {
		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge:               charge,
			Adapter:              s.adapter,
			Realizations:         s.realizations,
			Service:              s,
			CreditNotesSupported: s.creditNotesSupported.Load(),
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}

		err = stateMachine.FireAndActivate(ctx, patch.Trigger(), patch.TriggerParams())
		if err != nil {
			return nil, err
		}

		charge = stateMachine.GetCharge()
		result.InvoicePatches = stateMachine.DrainInvoicePatches()

		return &charge, nil
	})
	if err != nil {
		return meta.TriggerPatchResult[flatfee.Charge]{}, err
	}

	result.Charge = charge

	return result, nil
}

func (s *service) newStateMachine(config StateMachineConfig) (StateMachine, error) {
	switch config.Charge.Intent.SettlementMode {
	case productcatalog.CreditOnlySettlementMode:
		stateMachine, err := NewCreditsOnlyStateMachine(config)
		if err != nil {
			return nil, err
		}

		return stateMachine, nil
	case productcatalog.CreditThenInvoiceSettlementMode:
		stateMachine, err := NewCreditThenInvoiceStateMachine(config)
		if err != nil {
			return nil, err
		}

		return stateMachine, nil
	default:
		return nil, models.NewGenericNotImplementedError(
			fmt.Errorf("unsupported settlement mode %s for flat fee charge %s", config.Charge.Intent.SettlementMode, config.Charge.ID),
		)
	}
}

func (s *service) withLockedCharge(ctx context.Context, chargeID meta.ChargeID, fn func(ctx context.Context, charge flatfee.Charge) (*flatfee.Charge, error)) (*flatfee.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*flatfee.Charge, error) {
		key, err := charges.NewLockKeyForCharge(chargeID)
		if err != nil {
			return nil, fmt.Errorf("get charge lock key: %w", err)
		}

		if err := s.locker.LockForTX(ctx, key); err != nil {
			return nil, fmt.Errorf("lock charge: %w", err)
		}

		fetchedCharges, err := s.adapter.GetByIDs(ctx, flatfee.GetByIDsInput{
			Namespace: chargeID.Namespace,
			IDs:       []string{chargeID.ID},
			Expands:   meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("get charge: %w", err)
		}

		if len(fetchedCharges) == 0 {
			return nil, fmt.Errorf("charge not found [id=%s]", chargeID.ID)
		}

		charge := fetchedCharges[0]

		return fn(ctx, charge)
	})
}
