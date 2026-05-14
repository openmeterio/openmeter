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

		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("get currency calculator: %w", err)
		}

		stateMachine, err := s.newStateMachine(StateMachineConfig{
			Charge:             charge,
			Adapter:            s.adapter,
			Rater:              s.rater,
			Runs:               s.runs,
			CustomerOverride:   input.CustomerOverride,
			FeatureMeter:       featureMeter,
			CurrencyCalculator: currencyCalculator,
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
		stateMachineConfig, err := s.getStateMachineConfigForPatch(ctx, charge)
		if err != nil {
			return nil, fmt.Errorf("get state machine config: %w", err)
		}

		stateMachine, err := s.newStateMachine(stateMachineConfig)
		if err != nil {
			return nil, fmt.Errorf("new state machine: %w", err)
		}

		if err := stateMachine.FireAndActivate(ctx, patch.Trigger(), patch.TriggerParams()); err != nil {
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
			fmt.Errorf("unsupported settlement mode %s for usage based charge %s", config.Charge.Intent.SettlementMode, config.Charge.ID),
		)
	}
}

// getStateMachineConfigForPatch gets the state machine config for a patch.
//
// TODO[later]: This is something we can get from the callsite as we are doing a lot of unnecessary fetching here.
func (s *service) getStateMachineConfigForPatch(ctx context.Context, charge usagebased.Charge) (StateMachineConfig, error) {
	customerOverride, err := s.customerOverrideService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: charge.Namespace,
			ID:        charge.Intent.CustomerID,
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

	currencyCalculator, err := charge.Intent.Currency.Calculator()
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("get currency calculator: %w", err)
	}

	return StateMachineConfig{
		Charge:             charge,
		Adapter:            s.adapter,
		Rater:              s.rater,
		Runs:               s.runs,
		CustomerOverride:   customerOverride,
		FeatureMeter:       featureMeter,
		CurrencyCalculator: currencyCalculator,
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
