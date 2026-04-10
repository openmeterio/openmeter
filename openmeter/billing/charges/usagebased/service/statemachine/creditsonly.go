package statemachine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditsOnly struct {
	*Base
}

func NewCreditsOnly(config Config) (*CreditsOnly, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditOnlySettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_only", config.Charge.ID)
	}

	base, err := newBase(config)
	if err != nil {
		return nil, fmt.Errorf("new base: %w", err)
	}

	out := CreditsOnly{
		Base: base,
	}

	out.configureStates()

	return &out, nil
}

func (s *CreditsOnly) configureStates() {
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

func (s *CreditsOnly) ClearAdvanceAfter(ctx context.Context) error {
	s.Charge.State.AdvanceAfter = nil
	return nil
}

func (s *CreditsOnly) DeleteCharge(ctx context.Context, policy meta.PatchDeletePolicy) error {
	if policy.CreditRefundPolicy == meta.CreditRefundPolicyCorrect {
		currencyCalculator, err := s.Charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("get currency calculator: %w", err)
		}

		for _, run := range s.Charge.Realizations {
			realizationIDs := lo.Map(run.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
				return realization.ID
			})
			lineageSegmentsByRealization, err := s.Lineage.LoadActiveSegmentsByRealizationID(ctx, s.Charge.Namespace, realizationIDs)
			if err != nil {
				return fmt.Errorf("load active lineage segments for run %s: %w", run.ID.ID, err)
			}

			corrections, err := run.CreditsAllocated.CorrectAll(currencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.Handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
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
				if _, err := s.createRunCreditRealizations(ctx, s.Charge, run.ID, corrections); err != nil {
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

func (s *CreditsOnly) StartFinalRealizationRun(ctx context.Context) error {
	return s.Base.StartRealizationRun(ctx, StartRealizationRunInput{
		Type: usagebased.RealizationRunTypeFinalRealization,
	})
}
