package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	// Active

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerPartialInvoiceCreated,
			usagebased.StatusActivePartialInvoiceStarted,
		).
		Permit(
			meta.TriggerFinalInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			statelessx.AllOf(
				s.SyncFeatureIDFromFeatureMeter,
				s.AdvanceAfterServicePeriodTo,
			),
		)

	// ############################################################
	// Partial invoice realizations
	// ############################################################

	s.Configure(usagebased.StatusActivePartialInvoiceStarted).
		Permit(
			meta.TriggerFinalInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActivePartialInvoiceWaitingForCollection,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnEntry(statelessx.WithParameters(s.StartPartialInvoiceRun))

	s.Configure(usagebased.StatusActivePartialInvoiceWaitingForCollection).
		Permit(
			meta.TriggerFinalInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActivePartialInvoiceProcessing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActivePartialInvoiceProcessing).
		Permit(
			meta.TriggerFinalInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(
			meta.TriggerInvoiceIssued,
			usagebased.StatusActivePartialInvoiceIssuing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.SnapshotInvoiceUsage,
		)

	s.Configure(usagebased.StatusActivePartialInvoiceIssuing).
		Permit(
			meta.TriggerFinalInvoiceCreated,
			usagebased.StatusActiveFinalRealizationStarted,
			statelessx.BoolFn(s.IsAfterServicePeriod),
		).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActivePartialInvoiceCompleted,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.FinalizeInvoiceRun))

	s.Configure(usagebased.StatusActivePartialInvoiceCompleted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActive,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation))

	// Final (invoice) realizations

	s.Configure(usagebased.StatusActiveFinalRealizationStarted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationWaitingForCollection,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnEntry(statelessx.WithParameters(s.StartFinalInvoiceRun))

	s.Configure(usagebased.StatusActiveFinalRealizationWaitingForCollection).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActiveFinalRealizationProcessing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveFinalRealizationProcessing).
		Permit(
			meta.TriggerInvoiceIssued,
			usagebased.StatusActiveFinalRealizationIssuing,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.SnapshotInvoiceUsage,
		)

	s.Configure(usagebased.StatusActiveFinalRealizationIssuing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveFinalRealizationCompleted,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		// Extend is rejected while invoice-issued callbacks own this state.
		// Subscription sync can retry after billing advances the charge.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.FinalizeInvoiceRun))

	s.Configure(usagebased.StatusActiveFinalRealizationCompleted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveAwaitingPaymentSettlement,
		).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		// Extend is rejected because this branch still has its own next
		// transition to payment settlement. Subscription sync can retry.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation))

	// Payment + final

	s.Configure(usagebased.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerAllPaymentsSettled, usagebased.StatusFinal, statelessx.BoolFn(s.AreAllInvoicedRunsSettled)).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(usagebased.StatusFinal).
		Permit(meta.TriggerDelete, usagebased.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(usagebased.StatusDeleted).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, _ meta.PatchDeletePolicy) error {
	patches := []invoiceupdater.Patch{
		invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID),
	}

	for _, run := range s.Charge.Realizations {
		// Deleted realizations were already cleaned up through billing, so the
		// charge delete patch must not emit another invoice deletion for them.
		if run.DeletedAt != nil {
			continue
		}

		if run.LineID == nil || run.InvoiceID == nil {
			continue
		}

		patches = append(patches, invoiceupdater.NewDeleteLinePatch(
			billing.LineID{
				Namespace: s.Charge.Namespace,
				ID:        *run.LineID,
			},
			*run.InvoiceID,
		))
	}

	s.AddInvoicePatch(patches...)

	if err := s.Adapter.DeleteCharge(ctx, s.Charge); err != nil {
		return fmt.Errorf("delete charge: %w", err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("get charge: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) ExtendCharge(ctx context.Context, patch meta.PatchExtend) error {
	oldIntent := s.Charge.Intent.Intent
	if err := patch.ValidateWith(oldIntent); err != nil {
		return fmt.Errorf("validate extend patch: %w", err)
	}

	oldServicePeriod := meta.NormalizeClosedPeriod(s.Charge.Intent.ServicePeriod)

	s.Charge.Intent.ServicePeriod.To = patch.GetNewServicePeriodTo()
	s.Charge.Intent.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
	s.Charge.Intent.BillingPeriod.To = patch.GetNewBillingPeriodTo()
	s.Charge.Intent.InvoiceAt = patch.GetNewInvoiceAt()
	s.Charge.Intent = s.Charge.Intent.Normalized()

	newGatheringLinePeriod, err := s.handleFinalRunOnExtend(ctx, oldServicePeriod)
	if err != nil {
		return fmt.Errorf("handling final run on extend: %w", err)
	}

	if period, ok := newGatheringLinePeriod.Get(); ok {
		withLine, err := gatheringLineFromUsageBasedChargeForPeriod(s.Charge, period, s.Charge.Intent.InvoiceAt)
		if err != nil {
			return fmt.Errorf("creating gathering line for extended period: %w", err)
		}

		if withLine.GatheringLineToCreate == nil {
			return fmt.Errorf("creating gathering line for extended period: gathering line is required")
		}

		s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(*withLine.GatheringLineToCreate))
	} else {
		s.AddInvoicePatch(invoiceupdater.NewUpdateGatheringLineByChargeIDPatch(
			s.Charge.ID,
			s.Charge.Intent.ServicePeriod.To,
			s.Charge.Intent.InvoiceAt,
		))
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(_ context.Context, patch meta.PatchShrink) error {
	oldIntent := s.Charge.Intent.Intent
	if err := patch.ValidateWith(oldIntent); err != nil {
		return fmt.Errorf("validate shrink patch: %w", err)
	}

	s.Charge.Intent.ServicePeriod.To = patch.GetNewServicePeriodTo()
	s.Charge.Intent.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
	s.Charge.Intent.BillingPeriod.To = patch.GetNewBillingPeriodTo()
	s.Charge.Intent.InvoiceAt = patch.GetNewInvoiceAt()
	s.Charge.Intent = s.Charge.Intent.Normalized()

	if err := s.handleRunsOnShrink(); err != nil {
		return fmt.Errorf("handling realization runs on shrink: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) UnsupportedExtendOperation(_ context.Context, _ meta.PatchExtend) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot extend usage-based charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

func (s *CreditThenInvoiceStateMachine) UnsupportedShrinkOperation(_ context.Context, _ meta.PatchShrink) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot shrink usage-based charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

func (s *CreditThenInvoiceStateMachine) handleRunsOnShrink() error {
	newServicePeriodTo := meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.To)
	runsToKeep, runsToBeDeleted := s.Charge.Realizations.BisectByTimestamp(
		s.Charge.Intent.ServicePeriod,
		newServicePeriodTo,
	)

	for _, run := range runsToBeDeleted {
		if run.LineID == nil || run.InvoiceID == nil {
			return models.NewGenericPreConditionFailedError(
				fmt.Errorf("cannot shrink usage-based charge %s because realization run %s extends beyond the new service period and is not invoice-backed", s.Charge.ID, run.ID.ID),
			)
		}

		// Billing owns line cleanup. Mutable invoices can delete the line and
		// reverse draft effects; immutable invoices can keep history intact and
		// surface the missing prorating/credit-note work as validation.
		s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
			billing.LineID{
				Namespace: s.Charge.Namespace,
				ID:        *run.LineID,
			},
			*run.InvoiceID,
		))
	}

	gatheringLinePeriod := timeutil.ClosedPeriod{
		From: meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.From),
		To:   newServicePeriodTo,
	}

	for _, run := range runsToKeep {
		if run.IsVoidedBillingHistory() {
			continue
		}

		runServicePeriodTo := meta.NormalizeTimestamp(run.ServicePeriodTo)
		if runServicePeriodTo.After(gatheringLinePeriod.From) {
			gatheringLinePeriod.From = runServicePeriodTo
		}
	}

	gatheringLinePeriod = gatheringLinePeriod.Truncate(streaming.MinimumWindowSizeDuration)
	if len(runsToBeDeleted) == 0 && !gatheringLinePeriod.IsEmpty() {
		s.AddInvoicePatch(invoiceupdater.NewUpdateGatheringLineByChargeIDPatch(
			s.Charge.ID,
			gatheringLinePeriod.To,
			s.Charge.Intent.InvoiceAt,
		))
	} else {
		s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))

		if !gatheringLinePeriod.IsEmpty() {
			withLine, err := gatheringLineFromUsageBasedChargeForPeriod(
				s.Charge,
				gatheringLinePeriod,
				s.Charge.Intent.InvoiceAt,
			)
			if err != nil {
				return fmt.Errorf("creating gathering line for shrunk period: %w", err)
			}

			if withLine.GatheringLineToCreate == nil {
				return fmt.Errorf("creating gathering line for shrunk period: gathering line is required")
			}

			s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(*withLine.GatheringLineToCreate))
		}
	}

	if s.Charge.Status != usagebased.StatusCreated {
		s.Charge.Status = usagebased.StatusActive
		s.Charge.State.CurrentRealizationRunID = nil
		s.Charge.State.AdvanceAfter = lo.ToPtr(newServicePeriodTo)
	}

	return nil
}

// Extending a charge after a final invoice run moves the customer's contractual
// end date past a boundary that billing may have already turned into an invoice.
// Before invoice issuing, mutable invoice lines can still be rebuilt so the next
// billing cycle sees one coherent extended period. Once issuing starts, invoice
// and ledger side effects are external financial records and must stay intact. In
// that case the old invoice remains the historical partial bill and only the
// extended tail is left for a future invoice.
func (s *CreditThenInvoiceStateMachine) handleFinalRunOnExtend(ctx context.Context, oldServicePeriod timeutil.ClosedPeriod) (mo.Option[timeutil.ClosedPeriod], error) {
	if usagebased.IsMutableFinalRealizationStatus(s.Charge.Status) {
		if s.Charge.State.CurrentRealizationRunID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current final realization run is required [charge_id=%s,status=%s]", s.Charge.ID, s.Charge.Status)
		}

		currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
		if err != nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("get current realization run: %w", err)
		}

		if currentRun.Type != usagebased.RealizationRunTypeFinalRealization {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current run must be final realization [charge_id=%s,status=%s,run_id=%s,type=%s]", s.Charge.ID, s.Charge.Status, currentRun.ID.ID, currentRun.Type)
		}

		if currentRun.LineID == nil || currentRun.InvoiceID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current final realization run must be invoice-backed [charge_id=%s,status=%s,run_id=%s]", s.Charge.ID, s.Charge.Status, currentRun.ID.ID)
		}

		// Billing's mutable-line deletion hook owns the cleanup: it reverses the
		// draft allocations, marks this run deleted, and moves the charge back to
		// active for the extended period.
		s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
			billing.LineID{
				Namespace: s.Charge.Namespace,
				ID:        *currentRun.LineID,
			},
			*currentRun.InvoiceID,
		))

		return mo.Some(s.Charge.Intent.ServicePeriod), nil
	}

	finalRuns := lo.Filter(s.Charge.Realizations, func(run usagebased.RealizationRun, _ int) bool {
		if run.Type != usagebased.RealizationRunTypeFinalRealization {
			return false
		}

		// Voided realizations no longer preserve invoice lifecycle state, so they
		// cannot be reclassified when an already-extended charge is patched again.
		if run.IsVoidedBillingHistory() {
			return false
		}

		return meta.NormalizeTimestamp(run.ServicePeriodTo).Equal(oldServicePeriod.To)
	})
	if len(finalRuns) == 0 {
		return mo.None[timeutil.ClosedPeriod](), nil
	}

	finalRun := lo.MaxBy(finalRuns, func(run usagebased.RealizationRun, latest usagebased.RealizationRun) bool {
		return run.CreatedAt.After(latest.CreatedAt)
	})

	updatedRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:   finalRun.ID,
		Type: mo.Some(usagebased.RealizationRunTypePartialInvoice),
	})
	if err != nil {
		return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("reclassify final realization run[%s] as partial invoice: %w", finalRun.ID.ID, err)
	}

	finalRun.RealizationRunBase = updatedRunBase
	if err := s.Charge.Realizations.SetRealizationRun(finalRun); err != nil {
		return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("update realization run in charge: %w", err)
	}

	s.Charge.Status = usagebased.StatusActive
	s.Charge.State.CurrentRealizationRunID = nil
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.To))

	return mo.Some(timeutil.ClosedPeriod{
		From: oldServicePeriod.To,
		To:   s.Charge.Intent.ServicePeriod.To,
	}), nil
}

type invoiceCreatedInput struct {
	LineID          string
	InvoiceID       string
	ServicePeriodTo time.Time
}

func (i invoiceCreatedInput) Validate() error {
	if i.LineID == "" {
		return fmt.Errorf("line id is required")
	}

	if i.InvoiceID == "" {
		return fmt.Errorf("invoice id is required")
	}

	if i.ServicePeriodTo.IsZero() {
		return fmt.Errorf("service period to is required")
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) startInvoiceCreatedRun(
	ctx context.Context,
	input invoiceCreatedInput,
	runType usagebased.RealizationRunType,
) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate invoice created input: %w", err)
	}

	storedAtLT := meta.NormalizeTimestamp(input.ServicePeriodTo)
	servicePeriodTo := storedAtLT
	if runType == usagebased.RealizationRunTypeFinalRealization {
		var err error
		storedAtLT, err = s.getFinalRunStoredAtLT()
		if err != nil {
			return fmt.Errorf("get stored at lt: %w", err)
		}
		servicePeriodTo = meta.NormalizeTimestamp(s.Charge.Intent.ServicePeriod.To)
	}

	result, err := s.Runs.CreateRatedRun(ctx, usagebasedrun.CreateRatedRunInput{
		Charge:             s.Charge,
		CustomerOverride:   s.CustomerOverride,
		FeatureMeter:       s.FeatureMeter,
		Type:               runType,
		StoredAtLT:         storedAtLT,
		ServicePeriodTo:    servicePeriodTo,
		LineID:             lo.ToPtr(input.LineID),
		InvoiceID:          lo.ToPtr(input.InvoiceID),
		CreditAllocation:   usagebasedrun.CreditAllocationAvailable,
		CurrencyCalculator: s.CurrencyCalculator,
	})
	if err != nil {
		return err
	}

	s.Charge = result.Charge
	return nil
}

func (s *CreditThenInvoiceStateMachine) StartPartialInvoiceRun(ctx context.Context, input invoiceCreatedInput) error {
	return s.startInvoiceCreatedRun(ctx, input, usagebased.RealizationRunTypePartialInvoice)
}

func (s *CreditThenInvoiceStateMachine) StartFinalInvoiceRun(ctx context.Context, input invoiceCreatedInput) error {
	return s.startInvoiceCreatedRun(ctx, input, usagebased.RealizationRunTypeFinalRealization)
}

func resolveInvoiceCreatedTrigger(charge usagebased.Charge, billedPeriod timeutil.ClosedPeriod) meta.Trigger {
	if meta.NormalizeTimestamp(billedPeriod.To).Equal(meta.NormalizeTimestamp(charge.Intent.ServicePeriod.To)) {
		return meta.TriggerFinalInvoiceCreated
	}

	return meta.TriggerPartialInvoiceCreated
}

func (s *CreditThenInvoiceStateMachine) AreAllInvoicedRunsSettled() bool {
	return areAllInvoicedRunsSettled(s.Charge)
}

func (s *CreditThenInvoiceStateMachine) SnapshotInvoiceUsage(ctx context.Context) error {
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

	reconcileResult, err := s.Runs.ReconcileCredits(ctx, usagebasedrun.ReconcileCreditRealizationsInput{
		Charge:             s.Charge,
		Run:                currentRun,
		AllocateAt:         storedAtLT,
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

	if err := s.Adapter.UpsertRunDetailedLines(ctx, s.Charge.GetChargeID(), currentRun.ID, ratingResult.DetailedLines); err != nil {
		return fmt.Errorf("upsert run detailed lines: %w", err)
	}
	currentRun.DetailedLines = mo.Some(ratingResult.DetailedLines)

	currentRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:                        currentRun.ID,
		StoredAtLT:                mo.Some(storedAtLT),
		MeteredQuantity:           mo.Some(ratingResult.Quantity),
		Totals:                    mo.Some(currentTotals),
		NoFiatTransactionRequired: mo.Some(currentTotals.Total.IsZero()),
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

func getRunForLine(charge usagebased.Charge, lineID string) (usagebased.RealizationRun, error) {
	for _, run := range charge.Realizations {
		// Deleted realizations are no longer valid targets for invoice lifecycle callbacks.
		if run.DeletedAt != nil {
			continue
		}

		if run.LineID != nil && *run.LineID == lineID {
			return run, nil
		}
	}

	return usagebased.RealizationRun{}, fmt.Errorf("realization run not found for line %s", lineID)
}
