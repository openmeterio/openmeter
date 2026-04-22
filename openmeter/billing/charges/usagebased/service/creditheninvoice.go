package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type CreditThenInvoiceStateMachine struct {
	*stateMachine
}

func NewCreditThenInvoiceStateMachine(config StateMachineConfig) (*CreditThenInvoiceStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		return nil, fmt.Errorf("charge %s is not credit_then_invoice", config.Charge.ID)
	}

	stateMachine, err := newStateMachineBase(config)
	if err != nil {
		return nil, fmt.Errorf("new state machine: %w", err)
	}

	out := CreditThenInvoiceStateMachine{
		stateMachine: stateMachine,
	}

	out.configureStates()

	return &out, nil
}

func (s *CreditThenInvoiceStateMachine) configureStates() {
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
			meta.TriggerInvoiceCreated,
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
		OnEntry(statelessx.WithParameters(s.StartInvoiceCreatedRun))

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActiveFinalRealizationProcessing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnActive(
			s.SnapshotInvoiceUsage,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(
			meta.TriggerInvoiceIssued,
			usagebased.StatusActivePaymentPending,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted)

	s.Configure(usagebased.StatusActivePaymentPending).
		Permit(meta.TriggerInvoicePaymentAuthorized, usagebased.StatusActiveAuthorized).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.FinalizeInvoiceRun))

	s.Configure(usagebased.StatusActiveAuthorized).
		Permit(meta.TriggerInvoicePaymentSettled, usagebased.StatusFinal).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnEntryFrom(meta.TriggerInvoicePaymentAuthorized, statelessx.WithParameters(s.RecordPaymentAuthorized))

	s.Configure(usagebased.StatusFinal).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		OnEntryFrom(meta.TriggerInvoicePaymentSettled, statelessx.WithParameters(s.RecordPaymentSettled))

	s.Configure(usagebased.StatusDeleted).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, _ meta.PatchDeletePolicy) error {
	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	return nil
}

type invoiceCreatedInput struct {
	LineID string
}

func (i invoiceCreatedInput) Validate() error {
	if i.LineID == "" {
		return fmt.Errorf("line id is required")
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) StartInvoiceCreatedRun(ctx context.Context, input invoiceCreatedInput) error {
	storedAtOffset := meta.NormalizeTimestamp(clock.Now())
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
		LineID:             lo.ToPtr(input.LineID),
		CreditAllocation:   usagebasedrun.CreditAllocationAvailable,
		CurrencyCalculator: s.CurrencyCalculator,
	})
	if err != nil {
		return err
	}

	s.Charge = result.Charge
	return nil
}

func (s *CreditThenInvoiceStateMachine) SnapshotInvoiceUsage(ctx context.Context) error {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	if err := s.ensureDetailedLinesLoadedForRating(ctx); err != nil {
		return err
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return fmt.Errorf("get current realization run: %w", err)
	}

	storedAtOffset := meta.NormalizeTimestamp(currentRun.CollectionEnd)

	ratingResult, err := s.Rater.GetDetailedLinesForUsage(ctx, usagebasedrating.GetDetailedLinesForUsageInput{
		Charge:         s.Charge,
		PriorRuns:      s.Charge.Realizations.Without(currentRun.ID),
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)

	reconcileResult, err := s.Runs.ReconcileCredits(ctx, usagebasedrun.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                currentRun,
		AllocateAt:         storedAtOffset,
		TargetAmount:       currentTotals.Total,
		CurrencyCalculator: s.CurrencyCalculator,
		ExactAllocation:    false,
	})
	if err != nil {
		return fmt.Errorf("reconcile lifecycle: %w", err)
	}

	currentRun.CreditsAllocated = append(currentRun.CreditsAllocated, reconcileResult.Realizations...)
	currentTotals.CreditsTotal = s.CurrencyCalculator.RoundToPrecision(currentRun.CreditsAllocated.Sum())
	currentTotals.Total = s.CurrencyCalculator.RoundToPrecision(currentTotals.Total.Sub(currentTotals.CreditsTotal))

	runDetailedLines, err := s.Runs.PersistRunDetailedLines(ctx, s.Charge, currentRun, ratingResult)
	if err != nil {
		return err
	}
	currentRun.DetailedLines = mo.Some(runDetailedLines)

	currentRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:         currentRun.ID,
		AsOf:       mo.Some(storedAtOffset),
		MeterValue: mo.Some(ratingResult.Quantity),
		Totals:     mo.Some(currentTotals),
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

func (s *CreditThenInvoiceStateMachine) FinalizeInvoiceRun(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if s.Charge.State.CurrentRealizationRunID == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.GetCurrentRealizationRun()
	if err != nil {
		return fmt.Errorf("get current realization run: %w", err)
	}

	accrueResult, err := s.Runs.BookAccruedInvoiceUsage(ctx, usagebasedrun.BookAccruedInvoiceUsageInput{
		Charge: s.Charge,
		Run:    currentRun,
		Line:   *input.Line,
	})
	if err != nil {
		return fmt.Errorf("accrue invoice usage: %w", err)
	}
	currentRun = accrueResult.Run

	if err := s.Charge.Realizations.SetRealizationRun(currentRun); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	s.Charge.State.CurrentRealizationRunID = nil
	s.Charge.State.AdvanceAfter = nil

	updatedChargeBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase)
	if err != nil {
		return fmt.Errorf("update charge: %w", err)
	}

	s.Charge.ChargeBase = updatedChargeBase

	return nil
}

func (s *CreditThenInvoiceStateMachine) RecordPaymentAuthorized(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	run, err := s.getRunForLine(input.Line.ID)
	if err != nil {
		return err
	}

	result, err := s.Runs.BookInvoicedPaymentAuthorized(ctx, usagebasedrun.BookInvoicedPaymentAuthorizedInput{
		Charge:  s.Charge,
		Run:     run,
		Invoice: input.Invoice,
		Line:    *input.Line,
	})
	if err != nil {
		return fmt.Errorf("authorize invoice payment: %w", err)
	}

	if err := s.Charge.Realizations.SetRealizationRun(result.Run); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) RecordPaymentSettled(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	run, err := s.getRunForLine(input.Line.ID)
	if err != nil {
		return err
	}

	result, err := s.Runs.SettleInvoicedPayment(ctx, usagebasedrun.SettleInvoicedPaymentInput{
		Charge:  s.Charge,
		Run:     run,
		Invoice: input.Invoice,
		Line:    *input.Line,
	})
	if err != nil {
		return fmt.Errorf("settle invoice payment: %w", err)
	}

	if err := s.Charge.Realizations.SetRealizationRun(result.Run); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) getRunForLine(lineID string) (usagebased.RealizationRun, error) {
	for _, run := range s.Charge.Realizations {
		if run.LineID != nil && *run.LineID == lineID {
			return run, nil
		}
	}

	return usagebased.RealizationRun{}, fmt.Errorf("realization run not found for line %s", lineID)
}
