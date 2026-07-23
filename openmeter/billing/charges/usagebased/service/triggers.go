package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) AdvanceCharge(ctx context.Context, input usagebased.AdvanceChargeInput) (*usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	return s.withLockedCharge(ctx, input.ChargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) {
		featureMeter, err := charge.ResolveFeatureMeter(input.FeatureMeters)
		if err != nil {
			return nil, fmt.Errorf("get feature meter: %w", err)
		}

		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge:             charge,
			Adapter:            s.adapter,
			Rater:              s.rater,
			Runs:               s.runs,
			CustomerOverride:   input.CustomerOverride,
			FeatureMeter:       featureMeter,
			CurrencyCalculator: charge.Intent.GetCurrency(),
			CostBasisResolver:  s.costbasisResolver,
		})
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}

		return stateMachine.AdvanceUntilStateStable(ctx)
	})
}

func (s *service) TriggerPatch(ctx context.Context, chargeID meta.ChargeID, patch meta.Patch) (meta.TriggerPatchResult[usagebased.Charge], error) {
	if err := patch.Validate(); err != nil {
		return meta.TriggerPatchResult[usagebased.Charge]{}, fmt.Errorf("patch: %w", err)
	}

	if err := chargeID.Validate(); err != nil {
		return meta.TriggerPatchResult[usagebased.Charge]{}, fmt.Errorf("chargeID: %w", err)
	}

	var result meta.TriggerPatchResult[usagebased.Charge]

	charge, err := s.withLockedCharge(ctx, chargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) {
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
				return nil, fmt.Errorf("updating usage based charge[%s] base intent: %w", chargeWithUpdatedBase.ID, err)
			}

			chargeWithUpdatedBase.ChargeBase = updatedChargeBase

			return chargeWithUpdatedBase, nil
		}

		stateMachine, err := s.newStateMachineForCharge(ctx, charge)
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}

		if err := stateMachine.FireAndActivate(ctx, patch.Trigger(), patch); err != nil {
			return nil, err
		}

		charge = stateMachine.GetCharge()
		result.InvoicePatches = stateMachine.DrainInvoicePatches()

		return &charge, nil
	})
	if err != nil {
		return meta.TriggerPatchResult[usagebased.Charge]{}, err
	}

	result.Charge = charge

	return result, nil
}

func applyBaseIntentPatchForOverriddenCharge(charge usagebased.Charge, patch meta.Patch) (*usagebased.Charge, error) {
	target, err := patch.GetTargetLayer(charge.Intent)
	if err != nil {
		return nil, fmt.Errorf("getting patch target layer: %w", err)
	}

	if target != meta.ChangeTargetBase || !charge.Intent.HasOverrideLayer() {
		return nil, nil
	}

	switch patch := patch.(type) {
	case meta.PatchDelete:
		if err := charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *usagebased.IntentMutableFields) error {
			deletedAt := clock.Now()
			fields.IntentDeletedAt = &deletedAt
			return nil
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

func mutateBaseIntentPeriodForOverriddenCharge(charge *usagebased.Charge, patch periodPatch) error {
	if err := charge.Intent.Mutate(meta.ChangeTargetBase, func(fields *usagebased.IntentMutableFields) error {
		if err := patch.ValidateWith(fields.IntentMutableFields); err != nil {
			return fmt.Errorf("validate %s patch: %w", patch.Op(), err)
		}

		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()

		return nil
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
			fmt.Errorf("unsupported settlement mode %s for usage based charge %s", config.Charge.Intent.GetSettlementMode(), config.Charge.ID),
		)
	}
}

func (s *service) newStateMachineForCharge(ctx context.Context, charge usagebased.Charge) (StateMachine, error) {
	stateMachineConfig, err := s.getStateMachineConfigForCharge(ctx, charge)
	if err != nil {
		return nil, fmt.Errorf("get state machine config: %w", err)
	}

	stateMachine, err := s.newStateMachine(stateMachineConfig)
	if err != nil {
		return nil, fmt.Errorf("new state machine: %w", err)
	}

	return stateMachine, nil
}

// getStateMachineConfigForCharge gets the state machine config for a charge.
//
// TODO[later]: This is something we can get from the callsite as we are doing a lot of unnecessary fetching here.
func (s *service) getStateMachineConfigForCharge(ctx context.Context, charge usagebased.Charge) (StateMachineConfig, error) {
	customerOverride, err := s.customerOverrideService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: charge.Namespace,
			ID:        charge.Intent.GetCustomerID(),
		},
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
		},
	})
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("get customer override: %w", err)
	}

	featureRef := charge.GetFeatureKeyOrID()
	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, charge.Namespace, featureRef)
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("resolve feature meters: %w", err)
	}

	featureMeter, err := charge.ResolveFeatureMeter(featureMeters)
	if err != nil {
		return StateMachineConfig{}, err
	}

	currency := charge.Intent.GetCurrency()

	return StateMachineConfig{
		Charge:             charge,
		Adapter:            s.adapter,
		Rater:              s.rater,
		Runs:               s.runs,
		CustomerOverride:   customerOverride,
		FeatureMeter:       featureMeter,
		CurrencyCalculator: currency,
		CostBasisResolver:  s.costbasisResolver,
	}, nil
}

func (s *service) withLockedCharge(ctx context.Context, chargeID meta.ChargeID, fn func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error)) (*usagebased.Charge, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*usagebased.Charge, error) {
		key, err := charges.NewLockKeyForCharge(chargeID)
		if err != nil {
			return nil, fmt.Errorf("get charge lock key: %w", err)
		}

		if err := s.locker.LockForTX(ctx, key); err != nil {
			return nil, fmt.Errorf("lock charge: %w", err)
		}

		charge, err := s.adapter.GetByID(ctx, usagebased.GetByIDInput{
			ChargeID: chargeID,
			Expands:  meta.Expands{meta.ExpandRealizations},
		})
		if err != nil {
			return nil, fmt.Errorf("get charge: %w", err)
		}

		return fn(ctx, charge)
	})
}
