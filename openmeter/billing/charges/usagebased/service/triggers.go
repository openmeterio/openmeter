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
		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return nil, fmt.Errorf("get currency calculator: %w", err)
		}

		stateMachine, err := NewCreditsOnlyStateMachine(StateMachineConfig{
			Charge:             charge,
			Service:            s,
			CustomerOverride:   input.CustomerOverride,
			FeatureMeter:       input.FeatureMeter,
			CurrencyCalculator: currencyCalculator,
		})
		if err != nil {
			return nil, fmt.Errorf("new credits only state machine: %w", err)
		}

		return stateMachine.AdvanceUntilStateStable(ctx)
	})
}

func (s *service) TriggerPatch(ctx context.Context, chargeID meta.ChargeID, patch meta.Patch) (*usagebased.Charge, error) {
	if err := patch.Validate(); err != nil {
		return nil, fmt.Errorf("patch: %w", err)
	}

	if err := chargeID.Validate(); err != nil {
		return nil, fmt.Errorf("chargeID: %w", err)
	}

	return s.withLockedCharge(ctx, chargeID, func(ctx context.Context, charge usagebased.Charge) (*usagebased.Charge, error) {
		stateMachineConfig, err := s.getStateMachineConfigForPatch(ctx, charge)
		if err != nil {
			return nil, fmt.Errorf("get state machine config: %w", err)
		}

		stateMachine, err := NewCreditsOnlyStateMachine(stateMachineConfig)
		if err != nil {
			return nil, fmt.Errorf("new credits only state machine: %w", err)
		}

		if err := stateMachine.FireAndActivate(ctx, patch.Trigger(), patch.TriggerParams()); err != nil {
			return nil, err
		}

		return &stateMachine.Charge, nil
	})
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

	featureMeters, err := s.featureService.ResolveFeatureMeters(ctx, charge.Namespace, []string{charge.Intent.FeatureKey})
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("resolve feature meters: %w", err)
	}

	featureMeter, err := featureMeters.Get(charge.Intent.FeatureKey, true)
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("get feature meter: %w", err)
	}

	currencyCalculator, err := charge.Intent.Currency.Calculator()
	if err != nil {
		return StateMachineConfig{}, fmt.Errorf("get currency calculator: %w", err)
	}

	return StateMachineConfig{
		Charge:             charge,
		Service:            s,
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

		if charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
			return nil, models.NewGenericNotImplementedError(
				fmt.Errorf("charge %s is not credit_only (settlement_mode=%s)", charge.ID, charge.Intent.SettlementMode),
			)
		}

		return fn(ctx, charge)
	})
}
