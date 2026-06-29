package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
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

	out := CreditsOnlyStateMachine{
		stateMachine: stateMachine,
	}

	out.configureStates()

	return &out, nil
}

func (s *CreditsOnlyStateMachine) configureStates() {
	s.Configure(usagebased.StatusCreated).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActive,
			statelessx.BoolFn(s.IsInsideServicePeriod),
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			statelessx.AllOf(
				s.SyncFeatureIDFromFeatureMeter,
				s.AdvanceAfterServicePeriodTo,
			),
		)

	s.Configure(usagebased.StatusActiveFinalRealizationStarted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationWaitingForCollection,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.StartFinalRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationProcessing,
			s.IsAfterCollectionPeriod,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		// TODO: Transition to a failed state if the collection period end is not set
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		OnActive(
			s.FinalizeRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusFinal,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(usagebased.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.ClearAdvanceAfter)
}

func (s *CreditsOnlyStateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnlyStateMachine) DeleteCharge(ctx context.Context, patch meta.PatchDelete) error {
	if err := s.Charge.Intent.Mutate(patch.GetTarget(), func(fields *usagebased.IntentMutableFields) {
		fields.IntentDeletedAt = lo.ToPtr(clock.Now())
	}); err != nil {
		return fmt.Errorf("mutating %s intent deleted at: %w", patch.GetTarget(), err)
	}

	if patch.GetTarget() == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// customer-facing credit allocations remain owned by the override.
		return nil
	}

	s.Charge.Status = usagebased.StatusDeleted

	if patch.GetPolicy().CreditRefundPolicy == meta.CreditRefundPolicyCorrect {
		for _, run := range s.Charge.Realizations {
			if _, err := s.Runs.CorrectAllCredits(ctx, usagebasedrun.CorrectAllCreditRealizationsInput{
				Charge:             s.Charge,
				Run:                run,
				AllocateAt:         run.ServicePeriodTo,
				CurrencyCalculator: s.CurrencyCalculator,
			}); err != nil {
				return fmt.Errorf("correct credits for run %s: %w", run.ID.ID, err)
			}
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

func (s *CreditsOnlyStateMachine) ExtendCharge(ctx context.Context, patch meta.PatchExtend) error {
	patchResult, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !patchResult.ShouldReconcile {
		return nil
	}

	if err := s.voidAllRuns(ctx); err != nil {
		return err
	}

	return s.persistActivePeriodPatch(ctx)
}

func (s *CreditsOnlyStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	patchResult, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !patchResult.ShouldReconcile {
		return nil
	}

	if err := s.voidAllRuns(ctx); err != nil {
		return err
	}

	return s.persistActivePeriodPatch(ctx)
}

type creditsOnlyApplyPeriodPatchResult struct {
	ShouldReconcile bool
}

func (s *CreditsOnlyStateMachine) applyPeriodPatch(patch periodPatch) (creditsOnlyApplyPeriodPatchResult, error) {
	targetIntent, err := s.Charge.Intent.GetIntentForTarget(patch.GetTarget())
	if err != nil {
		return creditsOnlyApplyPeriodPatchResult{}, fmt.Errorf("getting %s intent: %w", patch.GetTarget(), err)
	}

	if err := patch.ValidateWith(targetIntent.IntentMutableFields.IntentMutableFields); err != nil {
		return creditsOnlyApplyPeriodPatchResult{}, fmt.Errorf("validate %s patch: %w", patch.Op(), err)
	}

	if err := s.Charge.Intent.Mutate(patch.GetTarget(), func(fields *usagebased.IntentMutableFields) {
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
	}); err != nil {
		return creditsOnlyApplyPeriodPatchResult{}, fmt.Errorf("mutating %s intent: %w", patch.GetTarget(), err)
	}

	if patch.GetTarget() == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// customer-facing credit allocations remain owned by the override.
		return creditsOnlyApplyPeriodPatchResult{}, nil
	}

	return creditsOnlyApplyPeriodPatchResult{
		ShouldReconcile: true,
	}, nil
}

func (s *CreditsOnlyStateMachine) persistActivePeriodPatch(ctx context.Context) error {
	s.Charge.Status = usagebased.StatusActive
	s.Charge.State.CurrentRealizationRunID = nil
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To))

	updatedBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase)
	if err != nil {
		return fmt.Errorf("update charge after period patch: %w", err)
	}
	s.Charge.ChargeBase = updatedBase

	return nil
}

func (s *CreditsOnlyStateMachine) voidAllRuns(ctx context.Context) error {
	// Credit-only usage-based charges currently have one realization run for the
	// whole service period. Void every run until periodic reconciliation and
	// progressive "billing" are implemented for usage-based charges.
	for _, run := range s.Charge.Realizations {
		if run.IsVoidedBillingHistory() {
			continue
		}

		if _, err := s.voidRealizationRun(ctx, run); err != nil {
			return err
		}
	}

	return nil
}

func (s *CreditsOnlyStateMachine) voidRealizationRun(ctx context.Context, run usagebased.RealizationRun) (usagebased.RealizationRun, error) {
	if _, err := s.Runs.CorrectAllCredits(ctx, usagebasedrun.CorrectAllCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                run,
		AllocateAt:         run.ServicePeriodTo,
		CurrencyCalculator: s.CurrencyCalculator,
	}); err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("correct credits for run %s: %w", run.ID.ID, err)
	}

	runBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:        run.ID,
		DeletedAt: mo.Some(lo.ToPtr(clock.Now())),
	})
	if err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("void realization run %s: %w", run.ID.ID, err)
	}

	run.RealizationRunBase = runBase
	if err := s.Charge.Realizations.SetRealizationRun(run); err != nil {
		return usagebased.RealizationRun{}, fmt.Errorf("update voided realization run %s: %w", run.ID.ID, err)
	}

	return run, nil
}

func (s *CreditsOnlyStateMachine) StartFinalRealizationRun(ctx context.Context) error {
	storedAtLT, err := s.getFinalRunStoredAtLT()
	if err != nil {
		return fmt.Errorf("get stored at lt: %w", err)
	}

	result, err := s.Runs.CreateRatedRun(ctx, usagebasedrun.CreateRatedRunInput{
		Charge:                    s.Charge,
		CustomerOverride:          s.CustomerOverride,
		FeatureMeter:              s.FeatureMeter,
		Type:                      usagebased.RealizationRunTypeFinalRealization,
		StoredAtLT:                storedAtLT,
		ServicePeriodTo:           meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To),
		CreditAllocation:          usagebasedrun.CreditAllocationExact,
		CurrencyCalculator:        s.CurrencyCalculator,
		NoFiatTransactionRequired: true,
	})
	if err != nil {
		return err
	}

	s.Charge = result.Charge
	return nil
}

func (s *CreditsOnlyStateMachine) FinalizeRealizationRun(ctx context.Context) error {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return fmt.Errorf("get current realization run: %w", err)
	}

	storedAtLT := meta.NormalizeTimestamp(currentRun.StoredAtLT)

	ratingResult, err := s.Rater.GetDetailedRatingForUsage(ctx, usagebasedrating.GetDetailedRatingForUsageInput{
		Charge:          s.Charge,
		StoredAtLT:      storedAtLT,
		ServicePeriodTo: currentRun.ServicePeriodTo,
		Customer:        s.CustomerOverride,
		FeatureMeter:    s.FeatureMeter,
	})
	if err != nil {
		return fmt.Errorf("get detailed rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)
	targetCreditsTotal := currentTotals.Total

	if _, err := s.Runs.ReconcileCredits(ctx, usagebasedrun.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                currentRun,
		AllocateAt:         currentRun.ServicePeriodTo,
		TargetAmount:       targetCreditsTotal,
		CurrencyCalculator: s.CurrencyCalculator,
		ExactAllocation:    true,
	}); err != nil {
		return fmt.Errorf("reconcile lifecycle: %w", err)
	}

	currentTotals.CreditsTotal = currentTotals.CreditsTotal.Add(targetCreditsTotal)
	currentTotals.Total = alpacadecimal.Zero

	if err := s.Adapter.UpsertRunDetailedLines(ctx, s.Charge.GetChargeID(), currentRun.ID, ratingResult.DetailedLines); err != nil {
		return fmt.Errorf("upsert run detailed lines: %w", err)
	}
	currentRun.DetailedLines = mo.Some(ratingResult.DetailedLines)

	currentRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:                        currentRun.ID,
		StoredAtLT:                mo.Some(storedAtLT),
		MeteredQuantity:           mo.Some(ratingResult.Quantity),
		Totals:                    mo.Some(currentTotals),
		NoFiatTransactionRequired: mo.Some(true),
	})
	if err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}
	currentRun.RealizationRunBase = currentRunBase

	if err := s.Charge.Realizations.SetRealizationRun(currentRun); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	s.Charge.State.CurrentRealizationRunID = nil
	if _, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase); err != nil {
		return fmt.Errorf("update charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch charge: %w", err)
	}

	return nil
}
