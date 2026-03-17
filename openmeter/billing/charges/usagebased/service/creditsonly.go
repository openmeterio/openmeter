package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

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
			usagebased.TriggerNext,
			usagebased.StatusActive,
			statelessx.BoolFn(s.IsInsideServicePeriod),
		).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	s.Configure(usagebased.StatusActive).
		Permit(
			usagebased.TriggerNext,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		OnActive(
			s.AdvanceAfterServicePeriodTo,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationStarted).
		Permit(
			usagebased.TriggerNext,
			usagebased.StatusActiveFinalRealizationWaitingForCollection,
		).
		OnActive(
			s.StartFinalRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			usagebased.TriggerNext,
			usagebased.StatusActiveFinalRealizationProcessing,
			s.IsAfterCollectionPeriod,
		).
		// TODO: Transition to a failed state if the collection period end is not set
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			usagebased.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		OnActive(
			s.FinalizeRealizationRun,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(
			usagebased.TriggerNext,
			usagebased.StatusFinal,
		)

	s.Configure(usagebased.StatusFinal).
		OnActive(s.ClearAdvanceAfter)
}

func (s *CreditsOnlyStateMachine) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnlyStateMachine) allocateCredits(ctx context.Context,
	in usagebased.AllocateCreditsInput,
	handlerFunc func(ctx context.Context, in usagebased.AllocateCreditsInput) (creditrealization.CreateInputs, error),
) (creditrealization.CreateInputs, error) {
	if err := in.Validate(); err != nil {
		return creditrealization.CreateInputs{}, err
	}

	creditAllocations, err := handlerFunc(ctx, in)
	if err != nil {
		return creditrealization.CreateInputs{}, fmt.Errorf("allocate credits: %w", err)
	}

	if in.Charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
		if !creditAllocations.Sum().Equal(in.AmountToAllocate) {
			return creditrealization.CreateInputs{}, usagebased.ErrCreditAllocationsDoNotMatchTotal.
				WithAttrs(models.Attributes{
					"total":     in.AmountToAllocate.String(),
					"charge_id": in.Charge.ID,
				})
		}
	}

	return creditAllocations, nil
}

func (s *CreditsOnlyStateMachine) StartFinalRealizationRun(ctx context.Context) error {
	storedAtOffset := clock.Now().Add(-usagebased.InternalCollectionPeriod)
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

	totals := ratingResult.Totals

	if totals.Total.IsNegative() {
		return usagebased.ErrChargeTotalIsNegative.
			WithAttrs(models.Attributes{
				"total":     totals.Total.String(),
				"charge_id": s.Charge.ID,
			})
	}

	var creditAllocations creditrealization.CreateInputs
	if !totals.Total.IsZero() {
		creditAllocations, err = s.allocateCredits(ctx,
			usagebased.AllocateCreditsInput{
				Charge:           s.Charge,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: totals.Total,
				CollectionType:   usagebased.RealizationRunTypeFinalRealization,
			},
			s.Service.handler.OnCollectionStarted,
		)
		if err != nil {
			return fmt.Errorf("allocate credits: %w", err)
		}
	}

	// We have allocated the required amount from credits, so we need to update totals accordingly
	totals.CreditsTotal = totals.CreditsTotal.Add(totals.Total)
	totals.Total = alpacadecimal.Zero

	updatedCharge, err := s.Service.createNewRealizationRun(ctx, s.Charge, createRealizationRunInput{
		Run: usagebased.CreateRealizationRunInput{
			Type:          usagebased.RealizationRunTypeFinalRealization,
			AsOf:          storedAtOffset,
			CollectionEnd: &collectionEnd,
			MeterValue:    ratingResult.Quantity,
			Totals:        totals,
		},
		CreditAllocations: creditAllocations,
	})
	if err != nil {
		return fmt.Errorf("create new realization run: %w", err)
	}

	s.Charge = updatedCharge

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

	if currentRun.CollectionEnd == nil {
		return fmt.Errorf("current realization run has no collection end [run_id=%s]", currentRun.ID.ID)
	}

	storedAtOffset := clock.Now().Add(-usagebased.InternalCollectionPeriod)

	ratingResult, err := s.Service.getRatingForUsage(ctx, getRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals
	currentTotals.CreditsTotal = currentTotals.CreditsTotal.Add(currentTotals.Total)
	currentTotals.Total = alpacadecimal.Zero

	additionalAmount := currentTotals.CreditsTotal.Sub(currentRun.Totals.CreditsTotal)

	switch {
	case additionalAmount.IsPositive():
		creditAllocations, err := s.allocateCredits(ctx,
			usagebased.AllocateCreditsInput{
				Charge:           s.Charge,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: additionalAmount,
				CollectionType:   usagebased.RealizationRunTypeFinalRealization,
			},
			s.Service.handler.OnCollectionFinalized,
		)
		if err != nil {
			return fmt.Errorf("allocate credits: %w", err)
		}

		if len(creditAllocations) > 0 {
			if _, err := s.Adapter.CreateRunCreditAllocations(ctx, currentRun.ID, creditAllocations); err != nil {
				return fmt.Errorf("create credit allocations: %w", err)
			}
		}
	case additionalAmount.IsNegative():
		return fmt.Errorf("unsupported: additional amount is negative [charge_id=%s, additional_amount=%s]", s.Charge.ID, additionalAmount.String())
	case additionalAmount.IsZero():
	}

	if _, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:            currentRun.ID,
		AsOf:          storedAtOffset,
		CollectionEnd: currentRun.CollectionEnd,
		MeterValue:    ratingResult.Quantity,
		Totals:        currentTotals,
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
