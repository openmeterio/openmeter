package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditsOnlyStateMachine struct {
	*stateMachine
}

func NewCreditsOnlyStateMachine(config StateMachineConfig) (*CreditsOnlyStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.GetSettlementMode() != productcatalog.CreditOnlySettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_only", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("new state machine: %w", err)
	}

	out := &CreditsOnlyStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *CreditsOnlyStateMachine) configureStates() {
	s.Configure(flatfee.StatusCreated).
		Permit(meta.TriggerNext, flatfee.StatusActive, statelessx.BoolFn(s.IsAfterInvoiceAt)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterInvoiceAt,
		)

	s.Configure(flatfee.StatusActive).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.IsAfterBookedAt)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterBookedAt,
		)

	s.Configure(flatfee.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			statelessx.AllOf(
				s.AllocateCredits,
				s.ClearAdvanceAfter,
			),
		)
}

func (s *CreditsOnlyStateMachine) IsAfterBookedAt() bool {
	return !clock.Now().Before(flatfee.UsageBookedAt(
		s.Charge.Intent.GetEffectivePaymentTerm(),
		s.Charge.Intent.GetEffectiveServicePeriod(),
	))
}

func (s *CreditsOnlyStateMachine) AdvanceAfterBookedAt(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(flatfee.UsageBookedAt(
		s.Charge.Intent.GetEffectivePaymentTerm(),
		s.Charge.Intent.GetEffectiveServicePeriod(),
	)))
	return nil
}

func (s *CreditsOnlyStateMachine) AllocateCredits(ctx context.Context) error {
	currencyCalculator, err := s.Charge.Intent.GetCurrency().Calculator()
	if err != nil {
		return fmt.Errorf("get currency calculator: %w", err)
	}

	amount := currencyCalculator.RoundToPrecision(s.Charge.State.AmountAfterProration)

	if amount.IsNegative() {
		return fmt.Errorf("charge total is negative [charge_id=%s, amount=%s]", s.Charge.ID, amount.String())
	}

	if s.Charge.Realizations.CurrentRun == nil {
		runBase, err := s.Adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
			Charge:                    s.Charge.ChargeBase,
			ServicePeriod:             s.Charge.Intent.GetEffectiveServicePeriod(),
			AmountAfterProration:      amount,
			NoFiatTransactionRequired: true, // We are in credits-only mode
		})
		if err != nil {
			return fmt.Errorf("create current run: %w", err)
		}

		s.Charge.Realizations.CurrentRun = &flatfee.RealizationRun{
			RealizationRunBase: runBase,
		}
	}

	if s.Charge.Realizations.CurrentRun != nil && len(s.Charge.Realizations.CurrentRun.CreditRealizations) > 0 {
		return s.reconcileCurrentRunCredits(ctx, amount)
	}

	result, err := s.Realizations.AllocateCreditsOnly(ctx, flatfeerealizations.AllocateCreditsOnlyInput{
		Charge:             s.Charge,
		Amount:             amount,
		CurrencyCalculator: currencyCalculator,
	})
	if err != nil {
		return fmt.Errorf("allocate credits: %w", err)
	}

	s.Charge.Realizations.CurrentRun.CreditRealizations = append(s.Charge.Realizations.CurrentRun.CreditRealizations, result.Realizations...)
	return nil
}

func (s *CreditsOnlyStateMachine) ExtendCharge(ctx context.Context, patch meta.PatchExtend) error {
	return s.applyPeriodPatch(ctx, patch)
}

func (s *CreditsOnlyStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	return s.applyPeriodPatch(ctx, patch)
}

func (s *CreditsOnlyStateMachine) applyPeriodPatch(ctx context.Context, patch periodPatch) error {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return err
	}

	targetIntent, err := s.Charge.Intent.GetIntentForTarget(target)
	if err != nil {
		return fmt.Errorf("getting %s intent: %w", target, err)
	}

	if err := patch.ValidateWith(targetIntent.IntentMutableFields.IntentMutableFields); err != nil {
		return fmt.Errorf("validate %s patch: %w", patch.Op(), err)
	}

	intent := s.Charge.Intent
	if err := intent.Mutate(target, func(fields *flatfee.IntentMutableFields) {
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
	}); err != nil {
		return fmt.Errorf("mutating %s intent: %w", target, err)
	}

	s.Charge.Intent = intent

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return fmt.Errorf("calculating amount after proration: %w", err)
	}
	s.Charge.State.AmountAfterProration = amountAfterProration

	if s.Charge.Realizations.CurrentRun == nil {
		return nil
	}

	return s.reconcileCurrentRunCredits(ctx, amountAfterProration)
}

func (s *CreditsOnlyStateMachine) reconcileCurrentRunCredits(ctx context.Context, amount alpacadecimal.Decimal) error {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return nil
	}

	currencyCalculator, err := s.Charge.Intent.GetCurrency().Calculator()
	if err != nil {
		return fmt.Errorf("get currency calculator: %w", err)
	}

	amount = currencyCalculator.RoundToPrecision(amount)
	servicePeriod := s.Charge.Intent.GetEffectiveServicePeriod()
	run := *currentRun
	run.ServicePeriod = servicePeriod

	reconcileResult, err := s.Realizations.ReconcileCredits(ctx, flatfeerealizations.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                run,
		AllocateAt:         flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), servicePeriod),
		TargetAmount:       amount,
		CurrencyCalculator: currencyCalculator,
	})
	if err != nil {
		return fmt.Errorf("reconcile credits for run %s: %w", run.ID.ID, err)
	}

	run.CreditRealizations = append(run.CreditRealizations, reconcileResult.Realizations...)

	runBase, err := s.Adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
		ID:                   run.ID,
		ServicePeriod:        mo.Some(servicePeriod),
		AmountAfterProration: mo.Some(amount),
		Totals: mo.Some(totals.Totals{
			Amount:       amount,
			CreditsTotal: amount,
			Total:        alpacadecimal.Zero,
		}),
		NoFiatTransactionRequired: mo.Some(true),
	})
	if err != nil {
		return fmt.Errorf("update credit-only run: %w", err)
	}

	run.RealizationRunBase = runBase
	s.Charge.Realizations.CurrentRun = &run
	return nil
}

func (s *CreditsOnlyStateMachine) DeleteCharge(ctx context.Context, patch meta.PatchDelete) error {
	deletedAt := lo.ToPtr(clock.Now())
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return err
	}

	if err := s.mutateIntentLayer(ctx, target, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = deletedAt
	}); err != nil {
		return fmt.Errorf("deleting intent: %w", err)
	}

	s.Charge.Status = flatfee.StatusDeleted

	if patch.GetPolicy().CreditRefundPolicy == meta.CreditRefundPolicyCorrect && s.Charge.Realizations.CurrentRun != nil {
		currencyCalculator, err := s.Charge.Intent.GetCurrency().Calculator()
		if err != nil {
			return fmt.Errorf("get currency calculator: %w", err)
		}

		if _, err := s.Realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             s.Charge,
			Run:                *s.Charge.Realizations.CurrentRun,
			AllocateAt:         flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), s.Charge.Realizations.CurrentRun.ServicePeriod),
			CurrencyCalculator: currencyCalculator,
		}); err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	return nil
}
