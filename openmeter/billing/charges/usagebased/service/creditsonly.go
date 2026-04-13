package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

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

	if config.Charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
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
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
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
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.StartFinalRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationProcessing,
			s.IsAfterCollectionPeriod,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		// TODO: Transition to a failed state if the collection period end is not set
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.FinalizeRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusFinal,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted)

	s.Configure(usagebased.StatusFinal).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(usagebased.StatusDeleted).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditsOnlyStateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnlyStateMachine) DeleteCharge(ctx context.Context, policy meta.PatchDeletePolicy) error {
	if policy.CreditRefundPolicy == meta.CreditRefundPolicyCorrect {
		for _, run := range s.Charge.Realizations {
			if _, err := s.Runs.CorrectAllCredits(ctx, usagebasedrun.CorrectAllCreditRealizationsInput{
				Charge:             s.Charge,
				Run:                run,
				AllocateAt:         clock.Now(),
				CurrencyCalculator: s.CurrencyCalculator,
			}); err != nil {
				return fmt.Errorf("correct credits for run %s: %w", run.ID.ID, err)
			}
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	return s.refetchCharge(ctx)
}

func (s *CreditsOnlyStateMachine) StartFinalRealizationRun(ctx context.Context) error {
	storedAtOffset := meta.NormalizeTimestamp(clock.Now().Add(-usagebased.InternalCollectionPeriod))
	collectionEnd, err := s.GetCollectionPeriodEnd(ctx)
	if err != nil {
		return fmt.Errorf("get collection period end: %w", err)
	}

	result, err := s.Runs.CreateRatedRun(ctx, usagebasedrun.CreateRatedRunInput{
		Charge:             s.Charge,
		CustomerOverride:   s.CustomerOverride,
		FeatureMeter:       s.FeatureMeter,
		Type:               usagebased.RealizationRunTypeFinalRealization,
		AsOf:               storedAtOffset,
		CollectionEnd:      collectionEnd,
		CreditAllocation:   usagebasedrun.CreditAllocationExact,
		CurrencyCalculator: s.CurrencyCalculator,
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

	storedAtOffset := meta.NormalizeTimestamp(clock.Now().Add(-usagebased.InternalCollectionPeriod))

	ratingResult, err := s.Rater.GetRatingForUsage(ctx, usagebasedrating.GetRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)
	targetCreditsTotal := currentTotals.Total

	if _, err := s.Runs.ReconcileCredits(ctx, usagebasedrun.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                currentRun,
		AllocateAt:         storedAtOffset,
		TargetAmount:       targetCreditsTotal,
		CurrencyCalculator: s.CurrencyCalculator,
		ExactAllocation:    true,
	}); err != nil {
		return fmt.Errorf("reconcile lifecycle: %w", err)
	}

	currentTotals.CreditsTotal = currentTotals.CreditsTotal.Add(targetCreditsTotal)
	currentTotals.Total = alpacadecimal.Zero

	if _, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:         currentRun.ID,
		AsOf:       storedAtOffset,
		MeterValue: ratingResult.Quantity,
		Totals:     currentTotals,
	}); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	s.Charge.State.CurrentRealizationRunID = nil
	if _, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase); err != nil {
		return fmt.Errorf("update charge: %w", err)
	}

	if err := s.refetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch charge: %w", err)
	}

	return nil
}
