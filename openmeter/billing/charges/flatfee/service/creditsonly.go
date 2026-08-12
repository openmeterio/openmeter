package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverride), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterInvoiceAt,
		)

	s.Configure(flatfee.StatusActive).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.IsAfterBookedAt)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverride), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterBookedAt,
		)

	s.Configure(flatfee.StatusFinal).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverride), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			statelessx.AllOf(
				s.AllocateCredits,
				s.ClearAdvanceAfter,
			),
		)

	s.Configure(flatfee.StatusDeleted).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverrideFromDeletedBase), statelessx.BoolFn(s.IsBaseIntentDeleted))

	s.Configure(flatfee.StatusActiveClearOverride).
		PermitDynamic(meta.TriggerNext, s.ResolveStateAfterClearOverride).
		OnActive(s.ActiveClearOverride)

	s.Configure(flatfee.StatusDeletedClearOverride).
		Permit(meta.TriggerNext, flatfee.StatusDeleted).
		OnActive(s.ClearDeletedChargeOverride)
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

func (s *CreditsOnlyStateMachine) SetOverride(ctx context.Context, patch flatfee.PatchSetOverride) error {
	if err := s.setOverrideIntent(ctx, patch); err != nil {
		return err
	}

	ratingResult, err := s.rateEffectiveIntent()
	if err != nil {
		return err
	}
	s.Charge.State.AmountAfterProration = ratingResult.Intent.AmountAfterProration

	if s.Charge.Realizations.CurrentRun == nil {
		return nil
	}

	return s.reconcileCurrentRunCredits(ctx, ratingResult)
}

func (s *CreditsOnlyStateMachine) ClearOverride(ctx context.Context, _ meta.PatchClearOverride) error {
	return s.ActiveClearOverride(ctx)
}

func (s *CreditsOnlyStateMachine) ActiveClearOverride(ctx context.Context) error {
	cleared, err := s.clearOverrideIntent(ctx)
	if err != nil {
		return err
	}
	if !cleared {
		return nil
	}
	if s.Charge.Intent.GetDeletedAt() != nil {
		return errors.New("clearing flat-fee override unexpectedly restored a deleted base intent")
	}

	ratingResult, err := s.rateEffectiveIntent()
	if err != nil {
		return err
	}
	s.Charge.State.AmountAfterProration = ratingResult.Intent.AmountAfterProration

	if s.Charge.Realizations.CurrentRun != nil {
		if err := s.reconcileCurrentRunCredits(ctx, ratingResult); err != nil {
			return err
		}
	}

	return nil
}

func (s *CreditsOnlyStateMachine) ResolveStateAfterClearOverride(_ context.Context, _ ...any) (stateless.State, error) {
	if s.IsAfterBookedAt() {
		return flatfee.StatusFinal, nil
	}

	if s.IsAfterInvoiceAt() {
		return flatfee.StatusActive, nil
	}

	return flatfee.StatusCreated, nil
}

func (s *CreditsOnlyStateMachine) ClearDeletedChargeOverride(ctx context.Context) error {
	cleared, err := s.clearOverrideIntent(ctx)
	if err != nil {
		return err
	}
	if !cleared {
		return nil
	}

	if s.Charge.Intent.GetDeletedAt() == nil {
		return errors.New("clearing flat-fee override did not restore a deleted base intent")
	}

	return s.reconcileDeletedCharge(ctx, meta.RefundAsCreditsDeletePolicy)
}

func (s *CreditsOnlyStateMachine) AllocateCredits(ctx context.Context) error {
	currency := s.Charge.Intent.GetCurrency()

	ratingResult, err := s.rateEffectiveIntent()
	if err != nil {
		return err
	}
	s.Charge.State.AmountAfterProration = ratingResult.Intent.AmountAfterProration

	if s.Charge.Realizations.CurrentRun == nil {
		runBase, err := s.Adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
			Charge:                    s.Charge.ChargeBase,
			ServicePeriod:             ratingResult.Intent.ServicePeriod,
			AmountAfterProration:      ratingResult.Intent.AmountAfterProration,
			NoFiatTransactionRequired: true, // We are in credits-only mode
		})
		if err != nil {
			return fmt.Errorf("create current run: %w", err)
		}

		s.Charge.Realizations.CurrentRun = &flatfee.RealizationRun{
			RealizationRunBase: runBase,
			DetailedLines:      mo.Some(ratingResult.DetailedLines),
		}

		if err := s.Adapter.UpsertDetailedLines(ctx, runBase.ID, ratingResult.DetailedLines); err != nil {
			return fmt.Errorf("persist credit-only detailed lines: %w", err)
		}
	}

	if s.Charge.Realizations.CurrentRun != nil && len(s.Charge.Realizations.CurrentRun.CreditRealizations) > 0 {
		return s.reconcileCurrentRunCredits(ctx, ratingResult)
	}

	result, err := s.Realizations.AllocateCreditsOnly(ctx, flatfeerealizations.AllocateCreditsOnlyInput{
		Charge:             s.Charge,
		Totals:             ratingResult.Totals,
		CurrencyCalculator: currency,
	})
	if err != nil {
		return fmt.Errorf("allocate credits: %w", err)
	}

	s.Charge.Realizations.CurrentRun.RealizationRunBase = result.RunBase
	s.Charge.Realizations.CurrentRun.DetailedLines = mo.Some(ratingResult.DetailedLines)
	s.Charge.Realizations.CurrentRun.CreditRealizations = append(s.Charge.Realizations.CurrentRun.CreditRealizations, result.CreditRealizations...)
	return nil
}

func (s *CreditsOnlyStateMachine) rateEffectiveIntent() (flatfeerealizations.RateResult, error) {
	rateableIntent, err := s.Charge.GetRateableIntent()
	if err != nil {
		return flatfeerealizations.RateResult{}, fmt.Errorf("getting rateable intent: %w", err)
	}

	ratingResult, err := s.Realizations.Rate(rateableIntent)
	if err != nil {
		return flatfeerealizations.RateResult{}, fmt.Errorf("rating flat fee: %w", err)
	}

	return ratingResult, nil
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

	ratingResult, err := s.rateEffectiveIntent()
	if err != nil {
		return err
	}
	s.Charge.State.AmountAfterProration = ratingResult.Intent.AmountAfterProration

	if s.Charge.Realizations.CurrentRun == nil {
		return nil
	}

	return s.reconcileCurrentRunCredits(ctx, ratingResult)
}

func (s *CreditsOnlyStateMachine) reconcileCurrentRunCredits(ctx context.Context, ratingResult flatfeerealizations.RateResult) error {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return nil
	}

	currency := s.Charge.Intent.GetCurrency()

	creditAllocationTarget := currency.RoundToPrecision(ratingResult.Totals.Total)
	servicePeriod := ratingResult.Intent.ServicePeriod
	run := *currentRun
	run.ServicePeriod = servicePeriod

	reconcileResult, err := s.Realizations.ReconcileCredits(ctx, flatfeerealizations.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                run,
		AllocateAt:         flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), servicePeriod),
		TargetAmount:       creditAllocationTarget,
		CurrencyCalculator: currency,
	})
	if err != nil {
		return fmt.Errorf("reconcile credits for run %s: %w", run.ID.ID, err)
	}

	run.CreditRealizations = append(run.CreditRealizations, reconcileResult.Realizations...)

	// Given ReconcileCredits is both used for credits only and credit-then-invoice modes,
	// we need to ensure that the allocated credits match the rated total.
	allocated := currency.RoundToPrecision(run.CreditRealizations.Sum())
	if !allocated.Equal(creditAllocationTarget) {
		return fmt.Errorf(
			"credit allocations do not match rated total [charge_id=%s total=%s allocated=%s]",
			s.Charge.ID,
			creditAllocationTarget.String(),
			allocated.String(),
		)
	}

	if err := s.Adapter.UpsertDetailedLines(ctx, run.ID, ratingResult.DetailedLines); err != nil {
		return fmt.Errorf("persist credit-only detailed lines: %w", err)
	}

	runTotals := ratingResult.Totals
	runTotals.CreditsTotal = currency.RoundToPrecision(runTotals.CreditsTotal.Add(allocated))
	runTotals.Total = currency.RoundToPrecision(runTotals.Total.Sub(allocated))

	runBase, err := s.Adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
		ID:                        run.ID,
		ServicePeriod:             mo.Some(servicePeriod),
		AmountAfterProration:      mo.Some(ratingResult.Intent.AmountAfterProration),
		Totals:                    mo.Some(runTotals),
		NoFiatTransactionRequired: mo.Some(true),
	})
	if err != nil {
		return fmt.Errorf("update credit-only run: %w", err)
	}

	run.RealizationRunBase = runBase
	run.DetailedLines = mo.Some(ratingResult.DetailedLines)
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
	if err := s.reconcileDeletedCharge(ctx, patch.GetPolicy()); err != nil {
		return err
	}

	return nil
}

func (s *CreditsOnlyStateMachine) reconcileDeletedCharge(ctx context.Context, policy meta.PatchDeletePolicy) error {
	if policy.CreditRefundPolicy == meta.CreditRefundPolicyCorrect && s.Charge.Realizations.CurrentRun != nil {
		currency := s.Charge.Intent.GetCurrency()

		if _, err := s.Realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             s.Charge,
			Run:                *s.Charge.Realizations.CurrentRun,
			AllocateAt:         flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), s.Charge.Realizations.CurrentRun.ServicePeriod),
			CurrencyCalculator: currency,
		}); err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch deleted charge: %w", err)
	}

	return nil
}
