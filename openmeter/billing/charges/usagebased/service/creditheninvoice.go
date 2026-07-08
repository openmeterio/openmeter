package service

import (
	"context"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"
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
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreditThenInvoiceStateMachine struct {
	*stateMachine
}

type periodPatch interface {
	Op() meta.PatchType
	GetTargetLayer(meta.LayeredIntentReader) (meta.ChangeTarget, error)
	GetNewServicePeriodTo() time.Time
	GetNewFullServicePeriodTo() time.Time
	GetNewBillingPeriodTo() time.Time
	GetNewInvoiceAt() time.Time
	ValidateWith(meta.IntentMutableFields) error
}

var (
	_ periodPatch = meta.PatchExtend{}
	_ periodPatch = meta.PatchShrink{}
)

func NewCreditThenInvoiceStateMachine(config StateMachineConfig) (*CreditThenInvoiceStateMachine, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	if config.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
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
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	// Active

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerInvoiceCreated,
			usagebased.StatusActiveRealizationStarted,
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

	// Invoice-backed realizations

	s.Configure(usagebased.StatusActiveRealizationStarted).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveRealizationWaitingForCollection,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnEntryFrom(meta.TriggerInvoiceCreated, statelessx.WithParameters(s.StartInvoiceRun))

	s.Configure(usagebased.StatusActiveRealizationWaitingForCollection).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActiveRealizationProcessing,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveRealizationProcessing).
		Permit(
			meta.TriggerInvoiceIssued,
			usagebased.StatusActiveRealizationIssuing,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(
			s.SnapshotInvoiceUsage,
		)

	s.Configure(usagebased.StatusActiveRealizationIssuing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveRealizationCompleted,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		// Extend is rejected while invoice-issued callbacks own this state.
		// Subscription sync can retry after billing advances the charge.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.FinalizeInvoiceRun))

	s.Configure(usagebased.StatusActiveRealizationCompleted).
		PermitDynamic(
			meta.TriggerNext,
			s.resolveStateAfterRealizationCompleted,
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		// Extend is rejected because this branch still has its own next
		// transition to payment settlement. Subscription sync can retry.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation))

	// Payment + final

	s.Configure(usagebased.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerAllPaymentsSettled, usagebased.StatusFinal, statelessx.BoolFn(s.AreAllInvoicedRunsSettled)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(usagebased.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(usagebased.StatusDeleted).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation))
}

func (s *CreditThenInvoiceStateMachine) resolveStateAfterRealizationCompleted(_ context.Context, _ ...any) (stateless.State, error) {
	latestRun, ok := s.Charge.Realizations.WithoutVoidedBillingHistory().Latest()
	if !ok {
		return nil, fmt.Errorf("no effective realization run found [charge_id=%s]", s.Charge.ID)
	}

	if isFinalRunInPeriod(s.Charge, timeutil.ClosedPeriod{
		From: s.Charge.Intent.GetEffectiveServicePeriod().From,
		To:   latestRun.ServicePeriodTo,
	}) {
		return usagebased.StatusActiveAwaitingPaymentSettlement, nil
	}

	return usagebased.StatusActive, nil
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, patch meta.PatchDelete) error {
	deletedAt := lo.ToPtr(clock.Now())
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.mutateIntentLayer(ctx, target, func(fields *usagebased.IntentMutableFields) {
		fields.IntentDeletedAt = deletedAt
	}); err != nil {
		return fmt.Errorf("deleting intent: %w", err)
	}

	if target == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// customer-facing invoice/run state remains owned by the override layer.
		return nil
	}

	s.Charge.Status = usagebased.StatusDeleted

	patches := invoiceupdater.Patches{
		invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID),
	}

	for _, run := range s.Charge.Realizations {
		// Voided realizations were already cleaned up through billing, so the
		// charge delete patch must not emit another invoice deletion for them.
		if run.IsVoidedBillingHistory() {
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
	patchResult, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !patchResult.ShouldReconcile {
		return nil
	}

	newGatheringLinePeriod, err := s.handleFinalRunOnExtend(ctx, patchResult.OldServicePeriod)
	if err != nil {
		return fmt.Errorf("handling final run on extend: %w", err)
	}

	if period, ok := newGatheringLinePeriod.Get(); ok {
		withLine, err := gatheringLineFromUsageBasedChargeForPeriod(s.Charge, period, s.Charge.Intent.GetEffectiveInvoiceAt())
		if err != nil {
			return fmt.Errorf("creating gathering line for extended period: %w", err)
		}

		if withLine.GatheringLineToCreate == nil {
			return fmt.Errorf("creating gathering line for extended period: gathering line is required")
		}

		s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(*withLine.GatheringLineToCreate))
	} else {
		gatheringLinePeriod := remainingGatheringLinePeriod(s.Charge)
		if gatheringLinePeriod.IsEmpty() {
			// Existing realization runs already cover the effective charge period,
			// so there is no remaining unbilled tail to keep as a pending line.
			s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))
			return nil
		}

		withLine, err := gatheringLineFromUsageBasedChargeForPeriod(
			s.Charge,
			gatheringLinePeriod,
			s.Charge.Intent.GetEffectiveInvoiceAt(),
		)
		if err != nil {
			return fmt.Errorf("creating gathering line update for extended period: %w", err)
		}

		if withLine.GatheringLineToCreate == nil {
			return fmt.Errorf("creating gathering line update for extended period: gathering line is required")
		}

		s.AddInvoicePatch(invoiceupdater.NewUpsertGatheringLineByChargeIDPatch(s.Charge.ID, *withLine.GatheringLineToCreate))
	}

	return nil
}

func remainingGatheringLinePeriod(charge usagebased.Charge) timeutil.ClosedPeriod {
	effectivePeriod := charge.Intent.GetEffectiveServicePeriod()
	period := timeutil.ClosedPeriod{
		From: meta.NormalizeTimestamp(effectivePeriod.From),
		To:   meta.NormalizeTimestamp(effectivePeriod.To),
	}

	for _, run := range charge.Realizations {
		if run.IsVoidedBillingHistory() {
			continue
		}

		runServicePeriodTo := meta.NormalizeTimestamp(run.ServicePeriodTo)
		if runServicePeriodTo.After(period.From) {
			period.From = runServicePeriodTo
		}
	}

	return period.Truncate(streaming.MinimumWindowSizeDuration)
}

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(_ context.Context, patch meta.PatchShrink) error {
	patchResult, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !patchResult.ShouldReconcile {
		return nil
	}

	if err := s.handleRunsOnShrink(); err != nil {
		return fmt.Errorf("handling realization runs on shrink: %w", err)
	}

	return nil
}

type creditThenInvoiceApplyPeriodPatchResult struct {
	ShouldReconcile  bool
	OldServicePeriod timeutil.ClosedPeriod
}

func (s *CreditThenInvoiceStateMachine) applyPeriodPatch(patch periodPatch) (creditThenInvoiceApplyPeriodPatchResult, error) {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("getting patch target layer: %w", err)
	}

	targetIntent, err := s.Charge.Intent.GetIntentForTarget(target)
	if err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("getting %s intent: %w", target, err)
	}

	if err := patch.ValidateWith(targetIntent.IntentMutableFields.IntentMutableFields); err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("validate %s patch: %w", patch.Op(), err)
	}

	oldServicePeriod := meta.NormalizeClosedPeriod(targetIntent.IntentMutableFields.ServicePeriod)

	if err := s.Charge.Intent.Mutate(target, func(fields *usagebased.IntentMutableFields) {
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
	}); err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("mutating %s intent: %w", target, err)
	}

	if target == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// customer-facing invoice/run state remains owned by the override layer.
		return creditThenInvoiceApplyPeriodPatchResult{}, nil
	}

	return creditThenInvoiceApplyPeriodPatchResult{
		ShouldReconcile:  true,
		OldServicePeriod: oldServicePeriod,
	}, nil
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
	servicePeriod := s.Charge.Intent.GetEffectiveServicePeriod()
	newServicePeriodTo := meta.NormalizeTimestamp(servicePeriod.To)
	runsToKeep, runsToBeDeleted := s.Charge.Realizations.BisectByTimestamp(
		servicePeriod,
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
		From: meta.NormalizeTimestamp(servicePeriod.From),
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
		withLine, err := gatheringLineFromUsageBasedChargeForPeriod(
			s.Charge,
			gatheringLinePeriod,
			s.Charge.Intent.GetEffectiveInvoiceAt(),
		)
		if err != nil {
			return fmt.Errorf("creating gathering line update for shrunk period: %w", err)
		}

		if withLine.GatheringLineToCreate == nil {
			return fmt.Errorf("creating gathering line update for shrunk period: gathering line is required")
		}

		s.AddInvoicePatch(invoiceupdater.NewUpsertGatheringLineByChargeIDPatch(s.Charge.ID, *withLine.GatheringLineToCreate))
	} else {
		s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))

		if !gatheringLinePeriod.IsEmpty() {
			withLine, err := gatheringLineFromUsageBasedChargeForPeriod(
				s.Charge,
				gatheringLinePeriod,
				s.Charge.Intent.GetEffectiveInvoiceAt(),
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

	s.updateStateAfterShrink(runsToKeep, gatheringLinePeriod, newServicePeriodTo)

	return nil
}

func (s *CreditThenInvoiceStateMachine) updateStateAfterShrink(
	runsToKeep usagebased.RealizationRuns,
	replacementGatheringLinePeriod timeutil.ClosedPeriod,
	newServicePeriodTo time.Time,
) {
	if s.Charge.Status == usagebased.StatusCreated {
		return
	}

	if s.Charge.State.CurrentRealizationRunID != nil {
		currentRunID := *s.Charge.State.CurrentRealizationRunID
		if _, err := runsToKeep.GetByID(currentRunID); err == nil {
			// Billing still owns the current invoice lifecycle. Shrink may shorten
			// future gathering, but it must not make the charge forget an in-flight
			// invoice-backed run that still fits inside the new service period.
			return
		}
	}

	s.Charge.State.CurrentRealizationRunID = nil

	if replacementGatheringLinePeriod.IsEmpty() && len(runsToKeep) > 0 {
		// The new service end is already covered by the last kept invoice-backed
		// run, so there is no future gathering work left for the charge. Decide
		// settlement from the kept effective history only; runs removed by this
		// shrink must not keep the charge waiting for callbacks that will never
		// arrive.
		chargeWithKeptRuns := s.Charge
		chargeWithKeptRuns.Realizations = runsToKeep
		if areAllInvoicedRunsSettled(chargeWithKeptRuns) {
			s.Charge.Status = usagebased.StatusFinal
		} else if s.Charge.Status != usagebased.StatusFinal {
			s.Charge.Status = usagebased.StatusActiveAwaitingPaymentSettlement
		}
		s.Charge.State.AdvanceAfter = nil

		return
	}

	s.Charge.Status = usagebased.StatusActive
	s.Charge.State.AdvanceAfter = lo.ToPtr(newServicePeriodTo)
}

// Extending a charge after a final invoice run moves the customer's contractual
// end date past a boundary that billing may have already turned into an invoice.
// Before invoice issuing, mutable invoice lines can still be rebuilt so the next
// billing cycle sees one coherent extended period. Once issuing starts, invoice
// and ledger side effects are external financial records and must stay intact. In
// that case the old invoice remains the historical partial bill and only the
// extended tail is left for a future invoice.
func (s *CreditThenInvoiceStateMachine) handleFinalRunOnExtend(ctx context.Context, oldServicePeriod timeutil.ClosedPeriod) (mo.Option[timeutil.ClosedPeriod], error) {
	if usagebased.IsMutableRealizationStatus(s.Charge.Status) {
		if s.Charge.State.CurrentRealizationRunID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current invoice-backed realization run is required [charge_id=%s,status=%s]", s.Charge.ID, s.Charge.Status)
		}

		currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
		if err != nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("get current realization run: %w", err)
		}

		if !meta.NormalizeTimestamp(currentRun.ServicePeriodTo).Equal(meta.NormalizeTimestamp(oldServicePeriod.To)) {
			return mo.None[timeutil.ClosedPeriod](), nil
		}

		if currentRun.LineID == nil || currentRun.InvoiceID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current terminal realization run must be invoice-backed [charge_id=%s,status=%s,run_id=%s]", s.Charge.ID, s.Charge.Status, currentRun.ID.ID)
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

		return mo.Some(s.Charge.Intent.GetEffectiveServicePeriod()), nil
	}

	finalRuns := lo.Filter(s.Charge.Realizations, func(run usagebased.RealizationRun, _ int) bool {
		// Voided realizations no longer preserve invoice lifecycle state, so they
		// cannot be reclassified when an already-extended charge is patched again.
		if run.IsVoidedBillingHistory() {
			return false
		}

		return meta.NormalizeTimestamp(run.ServicePeriodTo).Equal(meta.NormalizeTimestamp(oldServicePeriod.To))
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
	s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To))

	return mo.Some(timeutil.ClosedPeriod{
		From: oldServicePeriod.To,
		To:   s.Charge.Intent.GetEffectiveServicePeriod().To,
	}), nil
}

type invoiceCreatedInput struct {
	LineID        string
	InvoiceID     string
	ServicePeriod timeutil.ClosedPeriod
}

func (i invoiceCreatedInput) Validate() error {
	if i.LineID == "" {
		return fmt.Errorf("line id is required")
	}

	if i.InvoiceID == "" {
		return fmt.Errorf("invoice id is required")
	}

	if err := i.ServicePeriod.ValidateAsRequired(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) StartInvoiceRun(
	ctx context.Context,
	input invoiceCreatedInput,
) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validate invoice created input: %w", err)
	}

	runType := getInvoiceRealizationRunType(s.Charge, input.ServicePeriod)
	storedAtLT := meta.NormalizeTimestamp(input.ServicePeriod.To)
	servicePeriodTo := storedAtLT
	if runType == usagebased.RealizationRunTypeFinalRealization {
		var err error
		storedAtLT, err = s.getFinalRunStoredAtLT()
		if err != nil {
			return fmt.Errorf("get stored at lt: %w", err)
		}
		servicePeriodTo = meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To)
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

func getInvoiceRealizationRunType(charge usagebased.Charge, servicePeriod timeutil.ClosedPeriod) usagebased.RealizationRunType {
	if isFinalRunInPeriod(charge, servicePeriod) {
		return usagebased.RealizationRunTypeFinalRealization
	}

	return usagebased.RealizationRunTypePartialInvoice
}

func isFinalRunInPeriod(charge usagebased.Charge, servicePeriod timeutil.ClosedPeriod) bool {
	return meta.NormalizeTimestamp(servicePeriod.To).Equal(meta.NormalizeTimestamp(charge.Intent.GetEffectiveServicePeriod().To))
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
		AllocateAt:         currentRun.ServicePeriodTo,
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
		if run.LineID != nil && *run.LineID == lineID {
			return run, nil
		}
	}

	return usagebased.RealizationRun{}, fmt.Errorf("realization run not found for line %s", lineID)
}
