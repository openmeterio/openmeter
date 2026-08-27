package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qmuntal/stateless"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
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

var triggerSystemInvoiceLineDeleted meta.Trigger = "system_invoice_line_deleted"

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
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverride), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnActive(
			s.AdvanceAfterServicePeriodFrom,
		)

	// Active

	s.Configure(usagebased.StatusActive).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveAwaitingPaymentSettlement,
			statelessx.BoolFn(s.HasTerminalCompletedRealizationWithoutCurrentRun),
		).
		Permit(
			meta.TriggerInvoiceCreated,
			usagebased.StatusActiveRealizationStarted,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverride), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnActive(
			statelessx.AllOf(
				s.ResolveDynamicCostBasis,
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
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(triggerSystemInvoiceLineDeleted, statelessx.WithParameters(s.SystemInvoiceLineDeleted)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnEntryFrom(meta.TriggerInvoiceCreated, statelessx.WithParameters(s.StartInvoiceRun))

	s.Configure(usagebased.StatusActiveRealizationWaitingForCollection).
		Permit(
			meta.TriggerCollectionCompleted,
			usagebased.StatusActiveRealizationProcessing,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(triggerSystemInvoiceLineDeleted, statelessx.WithParameters(s.SystemInvoiceLineDeleted)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnActive(s.AdvanceAfterCollectionPeriodEnd)

	s.Configure(usagebased.StatusActiveRealizationProcessing).
		Permit(
			meta.TriggerNext,
			usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted,
			statelessx.BoolFn(s.IsCurrentRunZeroFiatAmountOverage),
		).
		Permit(
			meta.TriggerInvoiceFinalizing,
			usagebased.StatusActiveRealizationIssuing,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(triggerSystemInvoiceLineDeleted, statelessx.WithParameters(s.SystemInvoiceLineDeleted)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnActive(s.SnapshotInvoiceUsage)

	s.Configure(usagebased.StatusActiveRealizationIssuing).
		Permit(
			meta.TriggerInvoiceIssued,
			usagebased.StatusActiveRealizationCompleted,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(triggerSystemInvoiceLineDeleted, statelessx.WithParameters(s.SystemInvoiceLineDeleted)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		// Extend is rejected while invoice callbacks own this state.
		// Subscription sync can retry after billing advances the charge.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.UnsupportedShrinkToRealizedPeriodOperation)).
		OnEntryFrom(meta.TriggerInvoiceFinalizing, statelessx.WithParameters(s.FinalizeInvoice))

	// Zero-fiat-amount overages bypass invoice issuance after the run's converted
	// fiat amount is durable. This state finalizes that run before normal
	// post-realization routing.
	s.Configure(usagebased.StatusActiveRealizationZeroFiatAmountOverageCompleted).
		PermitDynamic(
			meta.TriggerNext,
			s.resolveStateAfterRealizationCompleted,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnActive(s.FinalizeZeroFiatAmountOverageRun)

	s.Configure(usagebased.StatusActiveRealizationCompleted).
		PermitDynamic(
			meta.TriggerNext,
			s.resolveStateAfterRealizationCompleted,
		).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		// Extend is rejected because this branch still has its own next
		// transition to payment settlement. Subscription sync can retry.
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		// Invoice-issued callbacks have already released the prepared run. A
		// gathering-line API delete may now shorten the
		// effective period to that completed run, and the next transition still
		// owns routing the charge to active or payment settlement.
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.InvoiceIssued)).
		OnActive(s.ReleaseInvoiceIssuedRun)

	// Payment + final

	s.Configure(usagebased.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerNext, usagebased.StatusFinal, statelessx.BoolFn(s.AreAllRealizationRunsSettled)).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod))

	s.Configure(usagebased.StatusFinal).
		Permit(meta.TriggerClearOverride, usagebased.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.UnsupportedClearOverrideOperation), statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.ShrinkToRealizedPeriod))

	s.Configure(usagebased.StatusDeleted).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, usagebased.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverrideFromDeletedBase), statelessx.BoolFn(s.IsBaseIntentDeleted)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerShrinkToRealizedPeriod, statelessx.WithParameters(s.UnsupportedShrinkToRealizedPeriodOperation))

	s.Configure(usagebased.StatusActiveClearOverride).
		PermitDynamic(meta.TriggerNext, s.ResolveStateAfterClearOverride).
		OnActive(s.ActiveClearOverride)

	s.Configure(usagebased.StatusDeletedClearOverride).
		Permit(meta.TriggerNext, usagebased.StatusDeleted).
		OnActive(s.ClearDeletedChargeOverride)
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

// HasTerminalCompletedRealizationWithoutCurrentRun lets the active state
// discover completed invoice-backed and zero-fiat overage realizations after
// manual period changes. Normally the realization branch moves to settlement
// when it creates the final run; API gathering-line deletes can instead make an
// existing partial run become final while the charge is already back in active.
func (s *CreditThenInvoiceStateMachine) HasTerminalCompletedRealizationWithoutCurrentRun() bool {
	if s.Charge.State.CurrentRealizationRunID != nil {
		return false
	}

	latestRun, ok := s.Charge.Realizations.WithoutVoidedBillingHistory().Latest()
	if !ok {
		return false
	}

	if latestRun.Type != usagebased.RealizationRunTypeFinalRealization {
		return false
	}

	if latestRun.InvoiceUsage == nil {
		fiatOverage, err := calculateFiatOverageForRun(s.Charge, latestRun)
		if err != nil || !fiatOverage.ShouldOmitInvoiceLine {
			// Stateless guards cannot return errors. Conversion failures are
			// exposed by line mapping or snapshotting before this state is durable.
			return false
		}
	}

	return isFinalRunInPeriod(s.Charge, timeutil.ClosedPeriod{
		From: s.Charge.Intent.GetEffectiveServicePeriod().From,
		To:   latestRun.ServicePeriodTo,
	})
}

func (s *CreditThenInvoiceStateMachine) SetOverride(ctx context.Context, patch usagebased.PatchSetOverride) error {
	if s.Charge.State.CurrentRealizationRunID != nil || len(s.Charge.Realizations.WithoutVoidedBillingHistory()) > 0 {
		// TODO: enable this once we have corrections and credit notes implemented.
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot set override for usage-based charge %s after realization has started", s.Charge.ID),
		)
	}

	if err := s.setOverrideIntent(ctx, patch); err != nil {
		return err
	}

	if err := s.updateGatheringLineForEffectiveIntent(); err != nil {
		return err
	}

	if s.Charge.Status == usagebased.StatusCreated {
		return s.AdvanceAfterServicePeriodFrom(ctx)
	}

	return s.AdvanceAfterServicePeriodTo(ctx)
}

func (s *CreditThenInvoiceStateMachine) ClearOverride(ctx context.Context, _ meta.PatchClearOverride) error {
	return s.ActiveClearOverride(ctx)
}

func (s *CreditThenInvoiceStateMachine) ActiveClearOverride(ctx context.Context) error {
	if s.Charge.State.CurrentRealizationRunID != nil || len(s.Charge.Realizations.WithoutVoidedBillingHistory()) > 0 {
		// TODO: enable this once we have corrections and credit notes implemented.
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot clear override for usage-based charge %s after realization has started", s.Charge.ID),
		)
	}

	if err := s.clearOverrideIntent(ctx); err != nil {
		return err
	}

	if s.Charge.Intent.GetBaseIntent().IntentDeletedAt != nil {
		return errors.New("clearing usage-based override unexpectedly restored a deleted base intent")
	}

	if err := s.updateGatheringLineForEffectiveIntent(); err != nil {
		return err
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) ResolveStateAfterClearOverride(_ context.Context, _ ...any) (stateless.State, error) {
	if clock.Now().Before(s.Charge.Intent.GetEffectiveServicePeriod().From) {
		return usagebased.StatusCreated, nil
	}

	return usagebased.StatusActive, nil
}

func (s *CreditThenInvoiceStateMachine) ClearDeletedChargeOverride(ctx context.Context) error {
	if err := s.clearOverrideIntent(ctx); err != nil {
		return err
	}

	if s.Charge.Intent.GetDeletedAt() == nil {
		return errors.New("clearing usage-based override did not restore a deleted base intent")
	}

	if err := s.reconcileDeletedCharge(ctx); err != nil {
		return err
	}

	return nil
}

// updateGatheringLineForEffectiveIntent keeps the pre-realization invoice
// placeholder aligned with the charge's currently effective mutable intent.
func (s *CreditThenInvoiceStateMachine) updateGatheringLineForEffectiveIntent() error {
	withLine, err := gatheringLineFromUsageBasedChargeForPeriod(
		s.Charge,
		s.Charge.Intent.GetEffectiveServicePeriod(),
		s.Charge.Intent.GetEffectiveInvoiceAt(),
	)
	if err != nil {
		return fmt.Errorf("creating gathering line for override: %w", err)
	}

	if withLine.GatheringLineToCreate == nil {
		return fmt.Errorf("creating gathering line for override: gathering line is required")
	}

	s.AddInvoicePatch(invoiceupdater.NewUpsertGatheringLineByChargeIDPatch(s.Charge.ID, *withLine.GatheringLineToCreate))

	return nil
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, patch meta.PatchDelete) error {
	deletedAt := lo.ToPtr(clock.Now())
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return err
	}

	if err := s.mutateIntentLayer(ctx, target, func(fields *usagebased.IntentMutableFields) error {
		fields.IntentDeletedAt = deletedAt
		return nil
	}); err != nil {
		return fmt.Errorf("deleting intent: %w", err)
	}

	s.Charge.Status = usagebased.StatusDeleted
	if err := s.reconcileDeletedCharge(ctx); err != nil {
		return err
	}

	return nil
}

// cancelRealizationRun owns charge-side cleanup for removing an invoice run.
// Reversible runs are corrected and marked deleted before their invoice patch
// is emitted; immutable and zero-fiat history is only detached from the current
// realization slot.
func (s *CreditThenInvoiceStateMachine) cancelRealizationRun(ctx context.Context, run usagebased.RealizationRun) error {
	shouldCorrect := false
	if !run.Immutable {
		if run.Payment != nil {
			return fmt.Errorf("cannot cancel realization run[%s] with payment allocation", run.ID.ID)
		}

		fiatOverage, err := calculateFiatOverageForRun(s.Charge, run)
		if err != nil {
			return fmt.Errorf("calculating fiat overage for realization run[%s]: %w", run.ID.ID, err)
		}
		shouldCorrect = !fiatOverage.ShouldOmitInvoiceLine
	}

	if shouldCorrect {
		correctionInput := usagebasedrun.CreditReconciliationHandlerInput{
			Charge:     s.Charge,
			Run:        run,
			AllocateAt: run.ServicePeriodTo,
		}
		if run.InvoiceUsage != nil {
			if !s.Charge.Intent.GetCurrency().IsCustom() {
				return fmt.Errorf("cannot cancel mutable regular-fiat realization run[%s] with invoice usage", run.ID.ID)
			}

			correctedRun, err := s.Runs.CorrectPreparedCustomCurrencyInvoiceRealizations(ctx, correctionInput)
			if err != nil {
				return fmt.Errorf("correcting prepared realization run[%s]: %w", run.ID.ID, err)
			}
			run = correctedRun
		} else if err := s.Runs.CorrectAllCreditRealizations(ctx, correctionInput); err != nil {
			return fmt.Errorf("correcting realization run[%s]: %w", run.ID.ID, err)
		}

		runBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
			ID:        run.ID,
			DeletedAt: mo.Some(lo.ToPtr(clock.Now())),
		})
		if err != nil {
			return fmt.Errorf("marking realization run[%s] deleted: %w", run.ID.ID, err)
		}
		run.RealizationRunBase = runBase
		s.Charge.Realizations = s.Charge.Realizations.Without(run.ID)
	}

	if run.LineID != nil && run.InvoiceID != nil {
		s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
			billing.LineID{Namespace: s.Charge.Namespace, ID: *run.LineID},
			*run.InvoiceID,
		))
	}

	currentRunCanceled := s.Charge.State.CurrentRealizationRunID != nil && *s.Charge.State.CurrentRealizationRunID == run.ID.ID
	if currentRunCanceled {
		s.Charge.State.CurrentRealizationRunID = nil
		if s.Charge.Status != usagebased.StatusDeleted {
			s.Charge.Status = usagebased.StatusActive
			s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(s.Charge.Intent.GetEffectiveServicePeriod().To))
		}

		updatedChargeBase, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase)
		if err != nil {
			return fmt.Errorf("detaching realization run[%s]: %w", run.ID.ID, err)
		}
		s.Charge.ChargeBase = updatedChargeBase
	}

	return nil
}

// SystemInvoiceLineDeleted reconciles a mutable invoice-backed run before
// billing completes a system-owned line deletion. The returned line patch is
// consumed by the callback because billing is already applying that exact
// effect; any gathering-line effect is forwarded back to billing.
func (s *CreditThenInvoiceStateMachine) SystemInvoiceLineDeleted(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	run, err := s.Charge.Realizations.GetByLineID(input.Line.ID)
	if err != nil {
		return fmt.Errorf("getting realization run for deleted invoice line[%s]: %w", input.Line.ID, err)
	}

	if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
		return fmt.Errorf("realization run[%s] is not associated with deleted invoice line[%s] on invoice[%s]", run.ID.ID, input.Line.ID, input.Invoice.ID)
	}

	if err := s.cancelRealizationRun(ctx, run); err != nil {
		return fmt.Errorf("canceling realization run[%s] for deleted invoice line[%s]: %w", run.ID.ID, input.Line.ID, err)
	}

	if input.Invoice.DeletionSource != "" {
		s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) reconcileDeletedCharge(ctx context.Context) error {
	s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))

	for _, run := range s.Charge.Realizations {
		// Voided realizations were already cleaned up through billing, so the
		// charge delete patch must not emit another invoice deletion for them.
		if run.IsVoidedBillingHistory() {
			continue
		}

		if err := s.cancelRealizationRun(ctx, run); err != nil {
			return fmt.Errorf("canceling realization run[%s] for deleted charge: %w", run.ID.ID, err)
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

func (s *CreditThenInvoiceStateMachine) ExtendCharge(ctx context.Context, patch meta.PatchExtend) error {
	patchResult, err := s.applyPeriodPatch(ctx, patch)
	if err != nil {
		return err
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

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	if _, err := s.applyPeriodPatch(ctx, patch); err != nil {
		return err
	}

	if err := s.handleRunsOnShrink(ctx); err != nil {
		return fmt.Errorf("handling realization runs on shrink: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) ShrinkToRealizedPeriod(ctx context.Context, patch meta.PatchShrinkToRealizedPeriod) error {
	if err := s.correctReversibleInvoicePreparation(ctx, patch.Op()); err != nil {
		return err
	}

	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return err
	}

	nonVoidedRuns := s.Charge.Realizations.WithoutVoidedBillingHistory()
	latestRun, ok := nonVoidedRuns.Latest()
	if !ok {
		return fmt.Errorf("cannot shrink usage-based charge %s to realized period without realization runs: %w", s.Charge.ID, billing.ErrCannotEditProgressivelyBilledUsageBasedLine)
	}
	newServicePeriodTo := meta.NormalizeTimestamp(patch.GetNewServicePeriodEnd())
	latestRunServicePeriodTo := meta.NormalizeTimestamp(latestRun.ServicePeriodTo)

	if err := s.mutateIntentLayer(ctx, target, func(fields *usagebased.IntentMutableFields) error {
		if err := patch.ValidateWith(fields.IntentMutableFields); err != nil {
			return fmt.Errorf("validate %s patch: %w", patch.Op(), err)
		}

		if !newServicePeriodTo.Equal(latestRunServicePeriodTo) {
			return fmt.Errorf(
				"cannot shrink usage-based charge %s to %s because latest realization run %s ends at %s: %w",
				s.Charge.ID,
				newServicePeriodTo,
				latestRun.ID.ID,
				latestRunServicePeriodTo,
				billing.ErrCannotEditProgressivelyBilledUsageBasedLine,
			)
		}

		fields.ServicePeriod.To = patch.GetNewServicePeriodEnd()
		return nil
	}); err != nil {
		return fmt.Errorf("mutating %s intent: %w", target, err)
	}

	if latestRun.Type == usagebased.RealizationRunTypePartialInvoice {
		updatedRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
			ID:   latestRun.ID,
			Type: mo.Some(usagebased.RealizationRunTypeFinalRealization),
		})
		if err != nil {
			return fmt.Errorf("updating realization run[%s] type: %w", latestRun.ID.ID, err)
		}

		latestRun.RealizationRunBase = updatedRunBase
		if err := s.Charge.Realizations.SetRealizationRun(latestRun); err != nil {
			return fmt.Errorf("updating in-memory realization run[%s]: %w", latestRun.ID.ID, err)
		}
	}

	s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))

	return nil
}

type creditThenInvoiceApplyPeriodPatchResult struct {
	OldServicePeriod timeutil.ClosedPeriod
}

func (s *CreditThenInvoiceStateMachine) applyPeriodPatch(ctx context.Context, patch periodPatch) (creditThenInvoiceApplyPeriodPatchResult, error) {
	if err := s.correctReversibleInvoicePreparation(ctx, patch.Op()); err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, err
	}

	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, err
	}

	var oldServicePeriod timeutil.ClosedPeriod

	if err := s.mutateIntentLayer(ctx, target, func(fields *usagebased.IntentMutableFields) error {
		if err := patch.ValidateWith(fields.IntentMutableFields); err != nil {
			return fmt.Errorf("validate %s patch: %w", patch.Op(), err)
		}

		oldServicePeriod = meta.NormalizeClosedPeriod(fields.ServicePeriod)
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
		return nil
	}); err != nil {
		return creditThenInvoiceApplyPeriodPatchResult{}, fmt.Errorf("mutating %s intent: %w", target, err)
	}

	return creditThenInvoiceApplyPeriodPatchResult{
		OldServicePeriod: oldServicePeriod,
	}, nil
}

// correctReversibleInvoicePreparation removes invoice-finalization effects so
// mutable usage-based intent changes never retain stale custom-currency state.
func (s *CreditThenInvoiceStateMachine) correctReversibleInvoicePreparation(ctx context.Context, op meta.PatchType) error {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return nil
	}

	currentRun, err := s.Charge.GetCurrentRealizationRun()
	if err != nil {
		return fmt.Errorf("getting current realization run before %s: %w", op, err)
	}
	if currentRun.Immutable || currentRun.InvoiceUsage == nil {
		return nil
	}

	if !s.Charge.Intent.GetCurrency().IsCustom() {
		return fmt.Errorf("cannot %s usage-based charge %s: mutable realization run %s has unexpected regular-fiat invoice usage", op, s.Charge.ID, currentRun.ID.ID)
	}

	if _, err := s.Runs.CorrectPreparedCustomCurrencyInvoiceRealizations(ctx, usagebasedrun.CreditReconciliationHandlerInput{
		Charge:     s.Charge,
		Run:        currentRun,
		AllocateAt: currentRun.ServicePeriodTo,
	}); err != nil {
		return fmt.Errorf("correct reversible realization run[%s] before %s: %w", currentRun.ID.ID, op, err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch usage-based charge[%s] after correcting realization run[%s]: %w", s.Charge.ID, currentRun.ID.ID, err)
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

func (s *CreditThenInvoiceStateMachine) UnsupportedShrinkToRealizedPeriodOperation(_ context.Context, _ meta.PatchShrinkToRealizedPeriod) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot shrink usage-based charge to realized period in status %s; retry after billing advances", s.Charge.Status),
	)
}

// ResolveDynamicCostBasis idempotently persists the dynamic cost basis
// effective at the charge's service-period start. Once resolved, the persisted
// value is authoritative and must never be overwritten by lifecycle retries.
func (s *CreditThenInvoiceStateMachine) ResolveDynamicCostBasis(ctx context.Context) error {
	intent := s.Charge.Intent.GetCostBasisIntent()
	if intent == nil || intent.Kind() != costbasis.ModeDynamic {
		return nil
	}

	if s.Charge.State.ResolvedCostBasis != nil {
		return nil
	}

	if s.Charge.State.CostBasisID == nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("dynamic cost basis reference is missing for usage based charge %s", s.Charge.ID),
		)
	}

	resolvedState, err := s.CostBasisResolver.ResolveDynamicState(ctx, costbasis.ResolveDynamicStateInput{
		CurrencyID:        s.Charge.Intent.GetCurrency().NamespacedID,
		Intent:            *intent,
		ServicePeriodFrom: s.Charge.Intent.GetEffectiveServicePeriod().From,
	})
	if err != nil {
		return fmt.Errorf("resolve dynamic cost basis for usage based charge %s: %w", s.Charge.ID, err)
	}

	persisted, err := s.Adapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
		NamespacedID: models.NamespacedID{
			Namespace: s.Charge.Namespace,
			ID:        *s.Charge.State.CostBasisID,
		},
		State: resolvedState,
	})
	if err != nil {
		return fmt.Errorf("persist dynamic cost basis for usage based charge %s: %w", s.Charge.ID, err)
	}

	if persisted.State == nil {
		return fmt.Errorf("persisted dynamic cost basis is unresolved for usage based charge %s", s.Charge.ID)
	}

	s.Charge.State.ResolvedCostBasis = persisted.State

	return nil
}

func (s *CreditThenInvoiceStateMachine) handleRunsOnShrink(ctx context.Context) error {
	servicePeriod := s.Charge.Intent.GetEffectiveServicePeriod()
	newServicePeriodTo := meta.NormalizeTimestamp(servicePeriod.To)
	runsToKeep, runsToBeDeleted, err := s.Charge.BisectRealizationRunsByTimestamp(newServicePeriodTo)
	if err != nil {
		return fmt.Errorf("bisecting usage-based realization runs: %w", err)
	}

	for _, run := range runsToBeDeleted {
		if run.LineID == nil || run.InvoiceID == nil {
			return models.NewGenericPreConditionFailedError(
				fmt.Errorf("cannot shrink usage-based charge %s because realization run %s extends beyond the new service period and is not invoice-backed", s.Charge.ID, run.ID.ID),
			)
		}

		if err := s.cancelRealizationRun(ctx, run); err != nil {
			return fmt.Errorf("canceling realization run[%s] after shrink: %w", run.ID.ID, err)
		}
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

	if err := s.updateStateAfterShrink(runsToKeep, gatheringLinePeriod, newServicePeriodTo); err != nil {
		return fmt.Errorf("updating state after shrink: %w", err)
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) updateStateAfterShrink(
	runsToKeep usagebased.RealizationRuns,
	replacementGatheringLinePeriod timeutil.ClosedPeriod,
	newServicePeriodTo time.Time,
) error {
	if s.Charge.Status == usagebased.StatusCreated {
		return nil
	}

	if s.Charge.State.CurrentRealizationRunID != nil {
		currentRunID := *s.Charge.State.CurrentRealizationRunID
		if _, err := runsToKeep.GetByID(currentRunID); err == nil {
			// Billing still owns the current invoice lifecycle. Shrink may shorten
			// future gathering, but it must not make the charge forget an in-flight
			// invoice-backed run that still fits inside the new service period.
			return nil
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
		allSettled, err := areAllRealizationRunsSettled(chargeWithKeptRuns)
		if err != nil {
			return err
		}
		if allSettled {
			s.Charge.Status = usagebased.StatusFinal
		} else if s.Charge.Status != usagebased.StatusFinal {
			s.Charge.Status = usagebased.StatusActiveAwaitingPaymentSettlement
		}
		s.Charge.State.AdvanceAfter = nil

		return nil
	}

	s.Charge.Status = usagebased.StatusActive
	s.Charge.State.AdvanceAfter = lo.ToPtr(newServicePeriodTo)

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
	if usagebased.IsMutableRealizationStatus(s.Charge.Status) {
		if s.Charge.State.CurrentRealizationRunID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current invoice-backed realization run is required [charge_id=%s,status=%s]", s.Charge.ID, s.Charge.Status)
		}

		currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
		if err != nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("get current realization run: %w", err)
		}

		// We don't want to delete a non-final realization run, just fall back to progressively billing the tail.
		if currentRun.Type != usagebased.RealizationRunTypeFinalRealization {
			return mo.None[timeutil.ClosedPeriod](), nil
		}

		if !meta.NormalizeTimestamp(currentRun.ServicePeriodTo).Equal(meta.NormalizeTimestamp(oldServicePeriod.To)) {
			return mo.None[timeutil.ClosedPeriod](), nil
		}

		if currentRun.LineID == nil || currentRun.InvoiceID == nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("current terminal realization run must be invoice-backed [charge_id=%s,status=%s,run_id=%s]", s.Charge.ID, s.Charge.Status, currentRun.ID.ID)
		}

		if err := s.cancelRealizationRun(ctx, currentRun); err != nil {
			return mo.None[timeutil.ClosedPeriod](), fmt.Errorf("canceling final realization run[%s] after extend: %w", currentRun.ID.ID, err)
		}

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

func (s *CreditThenInvoiceStateMachine) AreAllRealizationRunsSettled() bool {
	allSettled, err := areAllRealizationRunsSettled(s.Charge)
	if err != nil {
		// Stateless guards cannot return errors. Conversion failures are exposed
		// by line mapping or snapshotting before this transition becomes reachable.
		return false
	}

	return allSettled
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

	currentRun.StoredAtLT = storedAtLT
	reconciled, err := s.Runs.ReconcileRatedRun(ctx, usagebasedrun.ReconcileRatedRunInput{
		Charge:             s.Charge,
		Run:                currentRun,
		Rating:             ratingResult,
		CurrencyCalculator: s.CurrencyCalculator,
	})
	if err != nil {
		return fmt.Errorf("reconcile rated run: %w", err)
	}

	s.Charge = reconciled.Charge

	return nil
}

// IsCurrentRunZeroFiatAmountOverage reports whether the current run can
// complete without invoice issuance.
func (s *CreditThenInvoiceStateMachine) IsCurrentRunZeroFiatAmountOverage() bool {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return false
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return false
	}

	fiatOverage, err := calculateFiatOverageForRun(s.Charge, currentRun)
	if err != nil {
		// Stateless guards cannot return errors. Conversion failures are exposed
		// by line mapping or snapshotting before this transition becomes reachable.
		return false
	}

	return fiatOverage.ShouldOmitInvoiceLine
}

// FinalizeZeroFiatAmountOverageRun releases the persisted current run while the
// state machine is in zero-fiat-amount overage completion.
func (s *CreditThenInvoiceStateMachine) FinalizeZeroFiatAmountOverageRun(_ context.Context) error {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.GetCurrentRealizationRun()
	if err != nil {
		return fmt.Errorf("get current realization run: %w", err)
	}

	fiatOverage, err := calculateFiatOverageForRun(s.Charge, currentRun)
	if err != nil {
		return fmt.Errorf("calculate fiat overage for current realization run %s: %w", currentRun.ID.ID, err)
	}

	if !fiatOverage.ShouldOmitInvoiceLine {
		return fmt.Errorf("current realization run %s is not a zero fiat amount overage", currentRun.ID.ID)
	}

	s.Charge.State.CurrentRealizationRunID = nil
	s.Charge.State.AdvanceAfter = nil

	return nil
}

// FinalizeInvoice makes the line authoritative and performs reversible custom-
// currency accounting preparation before external invoice synchronization.
func (s *CreditThenInvoiceStateMachine) FinalizeInvoice(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
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
	if currentRun.LineID == nil || *currentRun.LineID != input.Line.ID {
		return fmt.Errorf("realization run[%s] line does not match finalizing line[%s]", currentRun.ID.ID, input.Line.ID)
	}
	if currentRun.InvoiceID == nil || *currentRun.InvoiceID != input.Invoice.ID {
		return fmt.Errorf("realization run[%s] invoice does not match finalizing invoice[%s]", currentRun.ID.ID, input.Invoice.ID)
	}

	if s.Charge.Intent.GetCurrency().IsCustom() {
		if currentRun.InvoiceUsage == nil && (len(currentRun.FiatOverageCreditRealizations) > 0 || currentRun.FiatOverageCreditAllocationCompleted) {
			return fmt.Errorf("realization run[%s] has fiat overage allocation state without prepared invoice usage", currentRun.ID.ID)
		}

		line, err := input.Line.Clone()
		if err != nil {
			return fmt.Errorf("cloning finalizing line[%s]: %w", input.Line.ID, err)
		}

		if currentRun.InvoiceUsage == nil {
			if err := populateStandardLineFromRun(line, populateStandardLineFromRunInput{
				Charge: s.Charge,
				Run:    currentRun,
				Stage:  standardLinePopulationStageInvoiceFinalizing,
			}); err != nil {
				return fmt.Errorf("populating gross finalizing line[%s] from run[%s]: %w", line.ID, currentRun.ID.ID, err)
			}

			prepared, err := s.Runs.BookAccruedInvoiceUsage(ctx, usagebasedrun.BookAccruedInvoiceUsageInput{
				Charge: s.Charge,
				Run:    currentRun,
				Line:   *line,
			})
			if err != nil {
				return fmt.Errorf("preparing custom-currency overage for finalizing line[%s]: %w", line.ID, err)
			}
			currentRun = prepared.Run
		}

		if !currentRun.FiatOverageCreditAllocationCompleted {
			allocated, err := s.Runs.AllocateFiatOverageCredits(ctx, usagebasedrun.AllocateFiatOverageCreditsInput{
				Charge: s.Charge,
				Run:    currentRun,
			})
			if err != nil {
				return fmt.Errorf("allocating fiat overage credits for finalizing line[%s]: %w", line.ID, err)
			}
			s.Charge = allocated.Charge
			currentRun = allocated.Run
		}

		if err := populateStandardLineFromRun(line, populateStandardLineFromRunInput{
			Charge: s.Charge,
			Run:    currentRun,
			Stage:  standardLinePopulationStageInvoiceFinalizing,
		}); err != nil {
			return fmt.Errorf("populating finalizing line[%s] from run[%s]: %w", line.ID, currentRun.ID.ID, err)
		}

		s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(line.AsGenericLine()))
	}

	s.Charge.State.AdvanceAfter = nil

	return nil
}

// InvoiceIssued accrues regular-fiat invoice usage or verifies custom-currency
// preparation, then marks the externally issued invoice history immutable.
func (s *CreditThenInvoiceStateMachine) InvoiceIssued(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
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
	if s.Charge.Intent.GetCurrency().IsCustom() {
		if currentRun.InvoiceUsage == nil {
			return fmt.Errorf("realization run[%s] has not been prepared for invoice issuance", currentRun.ID.ID)
		}
		if !currentRun.FiatOverageCreditAllocationCompleted {
			return fmt.Errorf("realization run[%s] has not completed fiat overage credit allocation", currentRun.ID.ID)
		}
	}
	if currentRun.LineID == nil || *currentRun.LineID != input.Line.ID {
		return fmt.Errorf("prepared realization run[%s] line does not match issued line[%s]", currentRun.ID.ID, input.Line.ID)
	}
	if currentRun.InvoiceID == nil || *currentRun.InvoiceID != input.Invoice.ID {
		return fmt.Errorf("prepared realization run[%s] invoice does not match issued invoice[%s]", currentRun.ID.ID, input.Invoice.ID)
	}

	if !s.Charge.Intent.GetCurrency().IsCustom() {
		accrueResult, err := s.Runs.BookAccruedInvoiceUsage(ctx, usagebasedrun.BookAccruedInvoiceUsageInput{
			Charge: s.Charge,
			Run:    currentRun,
			Line:   *input.Line,
		})
		if err != nil {
			return fmt.Errorf("accruing issued invoice usage: %w", err)
		}
		currentRun = accrueResult.Run
	}

	currentRun, err = s.Runs.MarkInvoiceIssued(ctx, currentRun)
	if err != nil {
		return fmt.Errorf("marking issued realization run[%s] immutable: %w", currentRun.ID.ID, err)
	}

	if err := s.Charge.Realizations.SetRealizationRun(currentRun); err != nil {
		return fmt.Errorf("updating issued realization run[%s]: %w", currentRun.ID.ID, err)
	}

	return nil
}

// ReleaseInvoiceIssuedRun clears lifecycle scheduling after the prepared run
// has accepted the external issuance event.
func (s *CreditThenInvoiceStateMachine) ReleaseInvoiceIssuedRun(_ context.Context) error {
	s.Charge.State.CurrentRealizationRunID = nil
	s.Charge.State.AdvanceAfter = nil

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
