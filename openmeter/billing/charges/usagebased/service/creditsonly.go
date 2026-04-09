package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditsOnlyStateMachine struct {
	*StateMachine
}

func NewCreditsOnlyStateMachine(config StateMachineConfig) (*CreditsOnlyStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_only", config.Charge.ID)
	}

	stateMachine, err := NewStateMachine(config)
	if err != nil {
		return nil, fmt.Errorf("new state machine: %w", err)
	}

	out := CreditsOnlyStateMachine{
		StateMachine: stateMachine,
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
		currencyCalculator, err := s.Charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("get currency calculator: %w", err)
		}

		for _, run := range s.Charge.Realizations {
			realizationIDs := lo.Map(run.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
				return realization.ID
			})
			lineageSegmentsByRealization, err := s.Service.lineage.LoadActiveSegmentsByRealizationID(ctx, s.Charge.Namespace, realizationIDs)
			if err != nil {
				return fmt.Errorf("load active lineage segments for run %s: %w", run.ID.ID, err)
			}

			corrections, err := run.CreditsAllocated.CorrectAll(currencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.Service.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
					Charge:                       s.Charge,
					Run:                          run,
					AllocateAt:                   clock.Now(),
					Corrections:                  req,
					LineageSegmentsByRealization: lineageSegmentsByRealization,
				})
			})
			if err != nil {
				return fmt.Errorf("correct credits for run %s: %w", run.ID.ID, err)
			}

			if len(corrections) > 0 {
				if _, err := s.Service.createRunCreditRealizations(ctx, s.Charge, run.ID, corrections); err != nil {
					return fmt.Errorf("create credit corrections for run %s: %w", run.ID.ID, err)
				}
			}
		}
	}

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	return s.refetchCharge(ctx)
}

func (s *CreditsOnlyStateMachine) allocateCredits(ctx context.Context, in usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateInputs, error) {
	in.AmountToAllocate = s.CurrencyCalculator.RoundToPrecision(in.AmountToAllocate)

	if err := in.Validate(); err != nil {
		return nil, err
	}

	creditAllocations, err := s.Service.handler.OnCreditsOnlyUsageAccrued(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	if !creditAllocations.Sum().Equal(in.AmountToAllocate) {
		return nil, usagebased.ErrCreditAllocationsDoNotMatchTotal.
			WithAttrs(models.Attributes{
				"total":     in.AmountToAllocate.String(),
				"charge_id": in.Charge.ID,
			})
	}

	return creditAllocations.AsCreateInputs(), nil
}

func (s *CreditsOnlyStateMachine) StartFinalRealizationRun(ctx context.Context) error {
	storedAtOffset := meta.NormalizeTimestamp(clock.Now().Add(-usagebased.InternalCollectionPeriod))
	collectionEnd, err := s.GetCollectionPeriodEnd(ctx)
	if err != nil {
		return fmt.Errorf("get collection period end: %w", err)
	}

	ratingResult, err := s.Service.getRatingForUsage(ctx, getRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	totals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)

	if totals.Total.IsNegative() {
		return usagebased.ErrChargeTotalIsNegative.
			WithAttrs(models.Attributes{
				"total":     totals.Total.String(),
				"charge_id": s.Charge.ID,
			})
	}

	updatedCharge, err := s.Service.createNewRealizationRun(ctx, s.Charge, usagebased.CreateRealizationRunInput{
		FeatureID:     s.Charge.State.FeatureID,
		Type:          usagebased.RealizationRunTypeFinalRealization,
		AsOf:          storedAtOffset,
		CollectionEnd: collectionEnd,
		MeterValue:    ratingResult.Quantity,
		Totals:        totals,
	})
	if err != nil {
		return fmt.Errorf("create new realization run: %w", err)
	}

	s.Charge = updatedCharge

	currentRun, err := updatedCharge.GetCurrentRealizationRun()
	if err != nil {
		return err
	}

	var creditRealizations creditrealization.Realizations
	if !totals.Total.IsZero() {
		creditAllocations, err := s.allocateCredits(ctx,
			usagebased.CreditsOnlyUsageAccruedInput{
				Charge:           updatedCharge,
				Run:              currentRun,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: totals.Total,
			},
		)
		if err != nil {
			return fmt.Errorf("allocate credits: %w", err)
		}

		if len(creditAllocations) > 0 {
			creditRealizations, err = s.Service.createRunCreditRealizations(ctx, updatedCharge, currentRun.ID, creditAllocations)
			if err != nil {
				return fmt.Errorf("create credit allocations: %w", err)
			}
		}

		currentRun.CreditsAllocated = creditRealizations
	}

	// We have allocated the required amount from credits, so we need to update totals accordingly
	totals.CreditsTotal = totals.CreditsTotal.Add(totals.Total)
	totals.Total = alpacadecimal.Zero

	currentRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:         currentRun.ID,
		AsOf:       storedAtOffset,
		MeterValue: ratingResult.Quantity,
		Totals:     totals,
	})
	if err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	currentRun.RealizationRunBase = currentRunBase

	if err := s.Charge.Realizations.SetRealizationRun(currentRun); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

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

	ratingResult, err := s.Service.getRatingForUsage(ctx, getRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)
	currentTotals.CreditsTotal = currentTotals.CreditsTotal.Add(currentTotals.Total)
	currentTotals.Total = alpacadecimal.Zero

	additionalAmount := s.CurrencyCalculator.RoundToPrecision(currentTotals.CreditsTotal.Sub(currentRun.Totals.CreditsTotal))

	switch {
	case additionalAmount.IsPositive():
		creditAllocations, err := s.allocateCredits(ctx,
			usagebased.CreditsOnlyUsageAccruedInput{
				Charge:           s.Charge,
				Run:              currentRun,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: additionalAmount,
			},
		)
		if err != nil {
			return fmt.Errorf("allocate credits: %w", err)
		}

		if len(creditAllocations) > 0 {
			if _, err := s.Service.createRunCreditRealizations(ctx, s.Charge, currentRun.ID, creditAllocations); err != nil {
				return fmt.Errorf("create credit allocations: %w", err)
			}
		}
	case additionalAmount.IsNegative():
		realizationIDs := lo.Map(currentRun.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
			return realization.ID
		})
		lineageSegmentsByRealization, err := s.Service.lineage.LoadActiveSegmentsByRealizationID(ctx, s.Charge.Namespace, realizationIDs)
		if err != nil {
			return fmt.Errorf("load active lineage segments for current run: %w", err)
		}

		corrections, err := currentRun.CreditsAllocated.Correct(
			additionalAmount,
			s.CurrencyCalculator,
			func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.Service.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
					Charge:                       s.Charge,
					Run:                          currentRun,
					AllocateAt:                   storedAtOffset,
					Corrections:                  req,
					LineageSegmentsByRealization: lineageSegmentsByRealization,
				})
			},
		)
		if err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}

		if len(corrections) > 0 {
			if _, err := s.Service.createRunCreditRealizations(ctx, s.Charge, currentRun.ID, corrections); err != nil {
				return fmt.Errorf("create credit corrections: %w", err)
			}
		}
	case additionalAmount.IsZero():
	}

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
