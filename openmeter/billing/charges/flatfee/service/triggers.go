package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
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
		chargeWithUpdatedBase, err := applyBaseIntentPatchForOverriddenCharge(charge, patch)
		if err != nil {
			return nil, err
		}

		if chargeWithUpdatedBase != nil {
			// Hidden base/source intent changes are subscription reconciliation,
			// not customer-facing lifecycle events. Persist the source intent and
			// skip the state machine because the active override owns lifecycle
			// state and hidden targets are rejected there.
			updatedChargeBase, err := s.adapter.UpdateCharge(ctx, chargeWithUpdatedBase.ChargeBase)
			if err != nil {
				return nil, fmt.Errorf("updating flat fee charge[%s] base intent: %w", chargeWithUpdatedBase.ID, err)
			}

			chargeWithUpdatedBase.ChargeBase = updatedChargeBase

			return chargeWithUpdatedBase, nil
		}

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

		err = stateMachine.FireAndActivate(ctx, patch.Trigger(), patch)
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

func applyBaseIntentPatchForOverriddenCharge(charge flatfee.Charge, patch meta.Patch) (*flatfee.Charge, error) {
	target, err := patch.GetTargetLayer(charge.Intent)
	if err != nil {
		return nil, fmt.Errorf("getting patch target layer: %w", err)
	}

	if target != meta.ChangeTargetBase || !charge.Intent.HasOverrideLayer() {
		return nil, nil
	}

	switch patch := patch.(type) {
	case meta.PatchDelete:
		if err := charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
			deletedAt := clock.Now()
			fields.IntentDeletedAt = &deletedAt
		}); err != nil {
			return nil, fmt.Errorf("mutating base intent for %s patch: %w", patch.Op(), err)
		}

		return &charge, nil
	case meta.PatchShrink:
		if err := mutateBaseIntentPeriodForOverriddenCharge(&charge, patch); err != nil {
			return nil, err
		}

		return &charge, nil
	case meta.PatchExtend:
		if err := mutateBaseIntentPeriodForOverriddenCharge(&charge, patch); err != nil {
			return nil, err
		}

		return &charge, nil
	}

	return nil, nil
}

func mutateBaseIntentPeriodForOverriddenCharge(charge *flatfee.Charge, patch periodPatch) error {
	targetIntent, err := charge.Intent.GetIntentForTarget(meta.ChangeTargetBase)
	if err != nil {
		return fmt.Errorf("getting base intent: %w", err)
	}

	if err := patch.ValidateWith(targetIntent.IntentMutableFields.IntentMutableFields); err != nil {
		return fmt.Errorf("validate %s patch: %w", patch.Op(), err)
	}

	if err := charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *flatfee.IntentMutableFields) {
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
	}); err != nil {
		return fmt.Errorf("mutating base intent for %s patch: %w", patch.Op(), err)
	}

	return nil
}

func (s *service) newStateMachine(config StateMachineConfig) (StateMachine, error) {
	switch config.Charge.Intent.GetSettlementMode() {
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
			fmt.Errorf("unsupported settlement mode %s for flat fee charge %s", config.Charge.Intent.GetSettlementMode(), config.Charge.ID),
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
