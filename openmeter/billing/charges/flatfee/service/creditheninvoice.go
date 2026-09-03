package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/qmuntal/stateless"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
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

	out := &CreditThenInvoiceStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *CreditThenInvoiceStateMachine) configureStates() {
	s.Configure(flatfee.StatusCreated).
		// Zero-amount CTI flat fees intentionally skip the billing line
		// engine. Once invoice_at is reached there will be no gathering
		// line to produce TriggerInvoiceCreated, so the charge closes
		// directly from created.
		Permit(
			meta.TriggerNext,
			flatfee.StatusFinal,
			statelessx.BoolFn(s.IsAfterInvoiceAtAndZeroAmount),
		).
		// Non-zero CTI flat fees become invoiceable at invoice_at. The line
		// engine creates the realization run from the standard invoice line,
		// which can happen before the service period starts for in-advance
		// flat fees.
		Permit(
			meta.TriggerNext,
			flatfee.StatusActive,
			statelessx.BoolFn(s.IsAfterInvoiceAtAndNonZeroAmount),
		).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		Permit(meta.TriggerAttachInvoiceLine, flatfee.StatusActiveRealizationProcessing).
		OnActive(s.AdvanceAfterInvoiceAt)

	s.Configure(flatfee.StatusActive).
		// This also repairs previously active zero-amount charges. They have
		// no line-engine path left, so active must not become their terminal
		// operational state.
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.IsZeroAmount)).
		Permit(meta.TriggerInvoiceCreated, flatfee.StatusActiveRealizationStarted).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnActive(statelessx.AllOf(
			s.ResolveDynamicCostBasis,
			s.AdvanceAfterServicePeriodTo,
		))

	s.Configure(flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationWaitingForCollection).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnEntryFrom(meta.TriggerInvoiceCreated, statelessx.WithParameters(s.StartRealization))

	s.Configure(flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerCollectionCompleted, flatfee.StatusActiveRealizationProcessing).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted)))

	s.Configure(flatfee.StatusActiveRealizationProcessing).
		Permit(
			meta.TriggerNext,
			flatfee.StatusActiveRealizationZeroFiatAmountOverageCompleted,
			statelessx.BoolFn(s.IsCurrentRunZeroFiatAmountOverage),
		).
		Permit(meta.TriggerInvoiceFinalizing, flatfee.StatusActiveRealizationIssuing).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnEntryFrom(meta.TriggerAttachInvoiceLine, statelessx.WithParameters(s.AttachInvoiceLine))

	s.Configure(flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerInvoiceIssued, flatfee.StatusActiveRealizationCompleted).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnEntryFrom(meta.TriggerInvoiceFinalizing, statelessx.WithParameters(s.FinalizeInvoice))

	// Zero-fiat-amount overages bypass invoice issuance. This state seals the
	// persisted current run before the charge completes without payment
	// settlement.
	s.Configure(flatfee.StatusActiveRealizationZeroFiatAmountOverageCompleted).
		Permit(meta.TriggerNext, flatfee.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnActive(s.FinalizeZeroFiatAmountOverageRun)

	s.Configure(flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerNext, flatfee.StatusActiveAwaitingPaymentSettlement).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.InvoiceIssued))

	s.Configure(flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted)))

	s.Configure(flatfee.StatusActiveClearOverride).
		PermitDynamic(meta.TriggerNext, s.ResolveStateAfterClearOverride).
		OnActive(s.ActiveClearOverride)

	s.Configure(flatfee.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.SetOverride)).
		Permit(meta.TriggerClearOverride, flatfee.StatusDeletedClearOverride, statelessx.BoolFn(s.IsBaseIntentDeleted)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation)).
		InternalTransition(meta.TriggerSetOverride, statelessx.WithParameters(s.UnsupportedSetOverrideOperation)).
		Permit(meta.TriggerClearOverride, flatfee.StatusActiveClearOverride, statelessx.BoolFn(statelessx.Not(s.IsBaseIntentDeleted))).
		InternalTransition(meta.TriggerClearOverride, statelessx.WithParameters(s.ClearOverrideFromDeletedBase), statelessx.BoolFn(s.IsBaseIntentDeleted))

	s.Configure(flatfee.StatusDeletedClearOverride).
		Permit(meta.TriggerNext, flatfee.StatusDeleted).
		OnActive(s.ClearDeletedChargeOverride)
}

func (s *CreditThenInvoiceStateMachine) SetOverride(ctx context.Context, patch flatfee.PatchSetOverride) error {
	oldAmountAfterProration := s.Charge.State.AmountAfterProration

	if err := s.setOverrideIntent(ctx, patch); err != nil {
		return err
	}

	amountAfterProration, err := s.Charge.Intent.CalculateAmountAfterProration()
	if err != nil {
		return fmt.Errorf("calculating amount after proration: %w", err)
	}

	return s.reconcileInvoicingState(ctx, reconcileInvoicingStateInput{
		Op:                      meta.PatchTypeSetOverride,
		Intent:                  s.Charge.Intent,
		OldAmountAfterProration: oldAmountAfterProration,
		NewAmountAfterProration: amountAfterProration,
	})
}

func (s *CreditThenInvoiceStateMachine) ActiveClearOverride(ctx context.Context) error {
	if !s.Charge.Intent.HasOverrideLayer() {
		return nil
	}

	if err := s.cancelCurrentRealization(ctx); err != nil {
		return err
	}

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

	amountAfterProration, err := s.Charge.Intent.CalculateAmountAfterProration()
	if err != nil {
		return fmt.Errorf("calculating amount after proration: %w", err)
	}

	s.Charge.State.AmountAfterProration = amountAfterProration
	if amountAfterProration.IsZero() {
		s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))
		return nil
	}

	gatheringLine, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
		Charge:    s.Charge,
		InvoiceAt: s.Charge.Intent.GetEffectiveInvoiceAt(),
	})
	if err != nil {
		return fmt.Errorf("creating gathering line after clearing override: %w", err)
	}

	s.AddInvoicePatch(invoiceupdater.NewUpsertGatheringLineByChargeIDPatch(s.Charge.ID, gatheringLine))
	return nil
}

// cancelCurrentRealization owns charge-side cleanup for removing the current
// invoice run. Reversible runs are corrected and marked deleted before their
// invoice patch is emitted; immutable and zero-fiat history is only detached.
func (s *CreditThenInvoiceStateMachine) cancelCurrentRealization(ctx context.Context) error {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return nil
	}
	run := *currentRun

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
		correctionInput := flatfeerealizations.CreditReconciliationHandlerInput{
			Charge:     s.Charge,
			Run:        run,
			AllocateAt: flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), run.ServicePeriod),
		}
		if run.AccruedUsage != nil {
			if !s.Charge.Intent.GetCurrency().IsCustom() {
				return fmt.Errorf("cannot cancel mutable regular-fiat realization run[%s] with invoice usage", run.ID.ID)
			}

			correctedRun, err := s.Realizations.CorrectPreparedCustomCurrencyInvoiceRealizations(ctx, correctionInput)
			if err != nil {
				return fmt.Errorf("correcting prepared realization run[%s]: %w", run.ID.ID, err)
			}
			run = correctedRun
		} else if err := s.Realizations.CorrectAllCreditRealizations(ctx, correctionInput); err != nil {
			return fmt.Errorf("correcting realization run[%s]: %w", run.ID.ID, err)
		}

		if err := s.Adapter.UpsertDetailedLines(ctx, run.ID, nil); err != nil {
			return fmt.Errorf("deleting detailed lines for realization run[%s]: %w", run.ID.ID, err)
		}

		runBase, err := s.Adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:        run.ID,
			DeletedAt: mo.Some(lo.ToPtr(clock.Now())),
		})
		if err != nil {
			return fmt.Errorf("marking realization run[%s] deleted: %w", run.ID.ID, err)
		}
		run.RealizationRunBase = runBase
	}

	if run.LineID != nil && run.InvoiceID != nil {
		s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
			billing.LineID{Namespace: s.Charge.Namespace, ID: *run.LineID},
			*run.InvoiceID,
		))
	}

	if err := s.Adapter.DetachCurrentRun(ctx, s.Charge.GetChargeID()); err != nil {
		return fmt.Errorf("detach current realization: %w", err)
	}

	s.Charge.Realizations.PriorRuns = append(s.Charge.Realizations.PriorRuns, run)
	s.Charge.Realizations.CurrentRun = nil

	return nil
}

func (s *CreditThenInvoiceStateMachine) ResolveStateAfterClearOverride(_ context.Context, _ ...any) (stateless.State, error) {
	if s.Charge.State.AmountAfterProration.IsZero() {
		return flatfee.StatusFinal, nil
	}

	if s.IsAfterInvoiceAt() {
		return flatfee.StatusActive, nil
	}

	return flatfee.StatusCreated, nil
}

func (s *CreditThenInvoiceStateMachine) ClearDeletedChargeOverride(ctx context.Context) error {
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

	return s.reconcileDeletedCharge(ctx)
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

	if err := s.mutateIntentLayer(ctx, target, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = deletedAt
	}); err != nil {
		return fmt.Errorf("deleting intent: %w", err)
	}

	s.Charge.Status = flatfee.StatusDeleted
	if err := s.reconcileDeletedCharge(ctx); err != nil {
		return err
	}

	return nil
}

func (s *CreditThenInvoiceStateMachine) reconcileDeletedCharge(ctx context.Context) error {
	s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))

	if err := s.cancelCurrentRealization(ctx); err != nil {
		return err
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
	invoicingStateInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	return s.reconcileInvoicingState(ctx, invoicingStateInput)
}

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	invoicingStateInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	return s.reconcileInvoicingState(ctx, invoicingStateInput)
}

func (s *CreditThenInvoiceStateMachine) LineManualEdit(ctx context.Context, patch meta.PatchLineManualEdit) error {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}
	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return err
	}

	override := patch.GetOverride()
	if err := meta.ValidateInvoiceLineOverrideDoesNotChangeImmutableChargeIntentFields(override); err != nil {
		return err
	}

	editedLine, err := override.ChangesToApply.Apply(override.ExistingLine)
	if err != nil {
		return fmt.Errorf("applying line manual edit: %w", err)
	}

	lineType := editedLine.AsInvoiceLine().Type()
	if chargeID := editedLine.GetChargeID(); chargeID == nil || *chargeID != s.Charge.ID {
		return fmt.Errorf("line[%s]: charge id must match flat-fee charge[%s]", editedLine.GetID(), s.Charge.ID)
	}

	switch lineType {
	case billing.InvoiceLineTypeGathering:
		if s.Charge.Realizations.CurrentRun != nil {
			return fmt.Errorf("partially-realized charge [charge_id=%s,run_id=%s]: %w",
				s.Charge.ID,
				s.Charge.Realizations.CurrentRun.ID.ID,
				billing.ErrCannotUpdateChargeManagedLine)
		}
	case billing.InvoiceLineTypeStandard:
		currentRun := s.Charge.Realizations.CurrentRun
		if currentRun == nil {
			return fmt.Errorf("missing current run [charge_id=%s,line_id=%s]: %w", s.Charge.ID, editedLine.GetID(), billing.ErrCannotUpdateChargeManagedLine)
		}

		if currentRun.Immutable {
			return fmt.Errorf("immutable current run [charge_id=%s,run_id=%s]: %w", s.Charge.ID, currentRun.ID.ID, billing.ErrCannotUpdateChargeManagedLine)
		}
		if currentRun.LineID == nil || *currentRun.LineID != editedLine.GetID() {
			return fmt.Errorf("run line mismatch [charge_id=%s,run_id=%s,line_id=%s,run_line_id=%s]: %w",
				s.Charge.ID,
				currentRun.ID.ID,
				editedLine.GetID(),
				lo.FromPtr(currentRun.LineID),
				billing.ErrCannotUpdateChargeManagedLine)
		}

		if currentRun.InvoiceID == nil || *currentRun.InvoiceID != editedLine.GetInvoiceID() {
			return fmt.Errorf("run invoice mismatch [charge_id=%s,run_id=%s,invoice_id=%s,run_invoice_id=%s]: %w",
				s.Charge.ID,
				currentRun.ID.ID,
				editedLine.GetInvoiceID(),
				lo.FromPtr(currentRun.InvoiceID),
				billing.ErrCannotUpdateChargeManagedLine)
		}
	default:
		return fmt.Errorf("unsupported line manual edit type [charge_id=%s,line_id=%s,line_type=%s]: %w",
			s.Charge.ID,
			editedLine.GetID(),
			lineType,
			billing.ErrCannotUpdateChargeManagedLine)
	}

	overrideFields, err := s.intentMutableFieldsFromLineManualEdit(editedLine)
	if err != nil {
		return fmt.Errorf("building intent override: %w", err)
	}

	oldAmountAfterProration := s.Charge.State.AmountAfterProration

	effectiveIntent := s.Charge.Intent.GetEffectiveIntent()
	effectiveIntent.IntentMutableFields = overrideFields
	amountAfterProration, err := effectiveIntent.CalculateAmountAfterProration()
	if err != nil {
		return fmt.Errorf("calculating amount after proration: %w", err)
	}

	if amountAfterProration.IsZero() {
		// TODO: support zero-proration manual line edits by modeling the API
		// result as a line deletion/detach instead of an updated line.
		// Until then, reject explicitly before persisting the override.
		return billing.ErrInvoiceLineZeroAmountDeleteInstead
	}

	if err := s.mutateIntentLayer(ctx, target, func(fields *flatfee.IntentMutableFields) {
		*fields = overrideFields
	}); err != nil {
		return fmt.Errorf("setting line manual edit intent: %w", err)
	}

	return s.reconcileInvoicingState(ctx, reconcileInvoicingStateInput{
		Op:                      meta.PatchTypeLineManualEdit,
		Intent:                  s.Charge.Intent,
		OldAmountAfterProration: oldAmountAfterProration,
		NewAmountAfterProration: amountAfterProration,
	})
}

func (s *CreditThenInvoiceStateMachine) applyPeriodPatch(patch periodPatch) (reconcileInvoicingStateInput, error) {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("getting patch target layer: %w", err)
	}
	if err := s.rejectHiddenIntentTarget(target); err != nil {
		return reconcileInvoicingStateInput{}, err
	}

	targetIntent, err := s.Charge.Intent.GetIntentForTarget(target)
	if err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("getting %s intent: %w", target, err)
	}

	if err := patch.ValidateWith(targetIntent.IntentMutableFields.IntentMutableFields); err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("validate %s patch: %w", patch.Op(), err)
	}
	intent := s.Charge.Intent
	if err := intent.Mutate(target, func(fields *flatfee.IntentMutableFields) {
		fields.ServicePeriod.To = patch.GetNewServicePeriodTo()
		fields.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
		fields.BillingPeriod.To = patch.GetNewBillingPeriodTo()
		fields.InvoiceAt = patch.GetNewInvoiceAt()
	}); err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("mutating %s intent: %w", target, err)
	}

	s.Charge.Intent = intent

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	return reconcileInvoicingStateInput{
		Op:                      patch.Op(),
		Intent:                  intent,
		OldAmountAfterProration: s.Charge.State.AmountAfterProration,
		NewAmountAfterProration: amountAfterProration,
	}, nil
}

func (s *CreditThenInvoiceStateMachine) UnsupportedExtendOperation(_ context.Context, _ meta.PatchExtend) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot extend flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
	)
}

func (s *CreditThenInvoiceStateMachine) UnsupportedShrinkOperation(_ context.Context, _ meta.PatchShrink) error {
	return models.NewGenericPreConditionFailedError(
		fmt.Errorf("cannot shrink flat-fee charge in status %s; retry after billing advances", s.Charge.Status),
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
			fmt.Errorf("dynamic cost basis reference is missing for flat-fee charge %s", s.Charge.ID),
		)
	}

	resolvedState, err := s.Service.costbasisResolver.ResolveDynamicState(ctx, costbasis.ResolveDynamicStateInput{
		CurrencyID:        s.Charge.Intent.GetCurrency().NamespacedID,
		Intent:            *intent,
		ServicePeriodFrom: s.Charge.Intent.GetEffectiveServicePeriod().From,
	})
	if err != nil {
		return fmt.Errorf("resolve dynamic cost basis for flat-fee charge %s: %w", s.Charge.ID, err)
	}

	persisted, err := s.Adapter.SetResolvedCostBasis(ctx, costbasis.SetResolvedCostBasisInput{
		NamespacedID: models.NamespacedID{
			Namespace: s.Charge.Namespace,
			ID:        *s.Charge.State.CostBasisID,
		},
		State: resolvedState,
	})
	if err != nil {
		return fmt.Errorf("persist dynamic cost basis for flat-fee charge %s: %w", s.Charge.ID, err)
	}

	if persisted.State == nil {
		return fmt.Errorf("persisted dynamic cost basis is unresolved for flat-fee charge %s", s.Charge.ID)
	}

	s.Charge.State.ResolvedCostBasis = persisted.State

	return nil
}

// StartRealization creates the current run. The line engine maps the run back
// onto the returned standard line before billing persists line updates.
func (s *CreditThenInvoiceStateMachine) StartRealization(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	run, err := s.Realizations.StartCreditThenInvoiceRun(ctx, flatfeerealizations.StartCreditThenInvoiceRunInput{
		Charge:    s.Charge,
		LineID:    input.Line.ID,
		InvoiceID: input.Invoice.ID,
	})
	if err != nil {
		return fmt.Errorf("start credit-then-invoice run: %w", err)
	}

	s.Charge.Realizations.CurrentRun = &run

	return nil
}

// AttachInvoiceLine turns a manually created charge into an invoice-backed
// charge by attaching its first realization run to the billing-preallocated
// standard line identity. The emitted patch is local to the API invoice edit
// flow: the line engine consumes it and returns the realized target state to
// billing instead of sending it through the subscription-sync invoice updater.
func (s *CreditThenInvoiceStateMachine) AttachInvoiceLine(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if s.Charge.Realizations.CurrentRun != nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot attach invoice line to flat-fee charge %s because current realization run %s already exists", s.Charge.ID, s.Charge.Realizations.CurrentRun.ID.ID),
		)
	}

	amountAfterProration, err := s.Charge.Intent.CalculateAmountAfterProration()
	if err != nil {
		return fmt.Errorf("calculating amount after proration: %w", err)
	}

	if amountAfterProration.IsZero() {
		return billing.ErrInvoiceLineZeroAmountCreate
	}

	gatheringLine, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
		Charge:    s.Charge,
		InvoiceAt: s.Charge.Intent.GetEffectiveInvoiceAt(),
	})
	if err != nil {
		return fmt.Errorf("creating flat-fee attach target line: %w", err)
	}

	line, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
	if err != nil {
		return fmt.Errorf("converting flat-fee attach target to standard line: %w", err)
	}

	line.ID = input.Line.ID

	run, err := s.Realizations.StartCreditThenInvoiceRun(ctx, flatfeerealizations.StartCreditThenInvoiceRunInput{
		Charge:    s.Charge,
		LineID:    line.ID,
		InvoiceID: input.Invoice.ID,
	})
	if err != nil {
		return fmt.Errorf("start attached credit-then-invoice run: %w", err)
	}

	s.Charge.Realizations.CurrentRun = &run

	if err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
		Charge: s.Charge,
		Run:    run,
		Stage:  standardLinePopulationStageManualAttachment,
	}); err != nil {
		return fmt.Errorf("mapping attached flat-fee run to standard line[%s]: %w", line.ID, err)
	}

	s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(line.AsGenericLine()))

	return nil
}

// FinalizeInvoice makes the line authoritative and performs reversible custom-
// currency accounting preparation before external invoice synchronization.
func (s *CreditThenInvoiceStateMachine) FinalizeInvoice(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}
	if currentRun.LineID == nil || *currentRun.LineID != input.Line.ID {
		return fmt.Errorf("realization run[%s] line does not match finalizing line[%s]", currentRun.ID.ID, input.Line.ID)
	}
	if currentRun.InvoiceID == nil || *currentRun.InvoiceID != input.Invoice.ID {
		return fmt.Errorf("realization run[%s] invoice does not match finalizing invoice[%s]", currentRun.ID.ID, input.Invoice.ID)
	}

	if !s.Charge.Intent.GetCurrency().IsCustom() {
		s.Charge.State.AdvanceAfter = nil

		return nil
	}

	run := *currentRun
	if run.AccruedUsage == nil && (len(run.FiatOverageCreditRealizations) > 0 || run.FiatOverageCreditAllocationCompleted) {
		return fmt.Errorf("realization run[%s] has fiat overage allocation state without prepared invoice usage", run.ID.ID)
	}

	line, err := input.Line.Clone()
	if err != nil {
		return fmt.Errorf("cloning finalizing line[%s]: %w", input.Line.ID, err)
	}

	if run.AccruedUsage == nil {
		if err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
			Charge: s.Charge,
			Run:    run,
			Stage:  standardLinePopulationStageInvoiceFinalizing,
		}); err != nil {
			return fmt.Errorf("populating gross finalizing line[%s] from run[%s]: %w", line.ID, run.ID.ID, err)
		}

		prepared, err := s.Realizations.AccrueInvoiceUsage(ctx, flatfeerealizations.AccrueInvoiceUsageInput{
			Charge: s.Charge,
			LineWithHeader: billing.StandardLineWithInvoiceHeader{
				Line:    line,
				Invoice: input.Invoice,
			},
		})
		if err != nil {
			return fmt.Errorf("preparing custom-currency overage for finalizing line[%s]: %w", line.ID, err)
		}
		run = prepared.Run
		s.Charge.Realizations.CurrentRun = &run
	}

	if !run.FiatOverageCreditAllocationCompleted {
		allocated, err := s.Realizations.AllocateFiatOverageCredits(ctx, flatfeerealizations.AllocateFiatOverageCreditsInput{
			Charge:           s.Charge,
			Run:              run,
			AllowFiatCredits: true,
		})
		if err != nil {
			return fmt.Errorf("allocating fiat overage credits for finalizing line[%s]: %w", line.ID, err)
		}
		run = allocated.Run
		s.Charge.Realizations.CurrentRun = &run
	}

	if err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
		Charge: s.Charge,
		Run:    run,
		Stage:  standardLinePopulationStageInvoiceFinalizing,
	}); err != nil {
		return fmt.Errorf("populating finalizing line[%s] from run[%s]: %w", line.ID, run.ID.ID, err)
	}

	s.Charge.State.AdvanceAfter = nil
	s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(line.AsGenericLine()))

	return nil
}

// InvoiceIssued accrues regular-fiat invoice usage or verifies custom-currency
// preparation, then marks the externally issued invoice history immutable.
func (s *CreditThenInvoiceStateMachine) InvoiceIssued(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	run := *currentRun
	if s.Charge.Intent.GetCurrency().IsCustom() {
		if currentRun.AccruedUsage == nil {
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
		result, err := s.Realizations.AccrueInvoiceUsage(ctx, flatfeerealizations.AccrueInvoiceUsageInput{
			Charge:         s.Charge,
			LineWithHeader: input,
		})
		if err != nil {
			return fmt.Errorf("accruing issued invoice usage: %w", err)
		}
		run = result.Run
	}

	run, err := s.Realizations.MarkInvoiceIssued(ctx, run)
	if err != nil {
		return fmt.Errorf("marking issued realization run[%s] immutable: %w", currentRun.ID.ID, err)
	}

	s.Charge.Realizations.CurrentRun = &run

	return nil
}

func (s *CreditThenInvoiceStateMachine) AreAllPaymentsSettled() bool {
	run := s.Charge.Realizations.CurrentRun
	if run == nil {
		return false
	}

	if run.AccruedUsage == nil || run.NoFiatTransactionRequired {
		return true
	}

	if run.Payment == nil {
		return false
	}

	return run.Payment.Status == payment.StatusSettled
}

// IsCurrentRunZeroFiatAmountOverage reports whether the current run can
// complete without invoice issuance.
func (s *CreditThenInvoiceStateMachine) IsCurrentRunZeroFiatAmountOverage() bool {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return false
	}

	fiatOverage, err := calculateFiatOverageForRun(s.Charge, *currentRun)
	if err != nil {
		// Stateless guards cannot return errors. Conversion failures are exposed
		// by line mapping before this transition becomes reachable.
		return false
	}

	return fiatOverage.ShouldOmitInvoiceLine
}

// FinalizeZeroFiatAmountOverageRun completes a run without crossing the
// external invoice issuance boundary.
func (s *CreditThenInvoiceStateMachine) FinalizeZeroFiatAmountOverageRun(_ context.Context) error {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	fiatOverage, err := calculateFiatOverageForRun(s.Charge, *currentRun)
	if err != nil {
		return fmt.Errorf("calculate fiat overage for current realization run %s: %w", currentRun.ID.ID, err)
	}

	if !fiatOverage.ShouldOmitInvoiceLine {
		return fmt.Errorf("current realization run %s is not a zero fiat amount overage", currentRun.ID.ID)
	}

	s.Charge.State.AdvanceAfter = nil

	return nil
}

type reconcileInvoicingStateInput struct {
	Op                      meta.PatchType
	Intent                  flatfee.OverridableIntent
	OldAmountAfterProration alpacadecimal.Decimal
	NewAmountAfterProration alpacadecimal.Decimal
}

func (s *CreditThenInvoiceStateMachine) reconcileInvoicingState(ctx context.Context, input reconcileInvoicingStateInput) error {
	if err := s.correctReversibleCurrentRun(ctx, input.Op); err != nil {
		return err
	}
	currentRun := s.Charge.Realizations.CurrentRun

	// TODO(credit-note support): this branch is a temporary fallback for
	// immutable invoice lines until the line updater can correct them with
	// credit notes. The normal patch flow below assumes immutable invoice
	// history can be adjusted safely; while that is false, we update the
	// charge intent/state but avoid creating replacement billable work for
	// the already-invoiced period.
	// A finalized zero-fiat run has immutable billing history, but deleting its
	// presentation line cannot require a credit note. Let the normal immutable
	// path detach that run and create replacement billable work.
	if !s.CreditNotesSupported && !s.IsCurrentRunZeroFiatAmountOverage() {
		// Case 1: We are trying to shrink an immutable invoice, but credit notes are not supported yet.

		// the immutable invoice cannot be corrected safely. Emit only the delete patch so the invoice
		// updater records an immutable-invoice warning; do not create replacement billable work for the
		// same already-invoiced period.
		//
		// This prevents charging both the non-prorated and prorated amounts.
		if currentRun != nil && currentRun.Immutable && !input.NewAmountAfterProration.Equal(input.OldAmountAfterProration) {
			if currentRun.LineID == nil {
				return models.NewGenericPreConditionFailedError(
					fmt.Errorf("cannot %s flat-fee charge %s because current realization run %s does not have a persisted line reference", input.Op, s.Charge.ID, currentRun.ID.ID),
				)
			}

			if currentRun.InvoiceID == nil {
				return models.NewGenericPreConditionFailedError(
					fmt.Errorf("cannot %s flat-fee charge %s because current realization run %s does not have a persisted invoice reference", input.Op, s.Charge.ID, currentRun.ID.ID),
				)
			}

			s.Charge.Intent = input.Intent
			s.Charge.State.AmountAfterProration = input.NewAmountAfterProration

			s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
				billing.LineID{
					Namespace: s.Charge.Namespace,
					ID:        *currentRun.LineID,
				},
				*currentRun.InvoiceID,
			))

			return nil
		}
	}

	s.Charge.Intent = input.Intent
	s.Charge.State.AmountAfterProration = input.NewAmountAfterProration

	updatedGatheringLine, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
		Charge:    s.Charge,
		InvoiceAt: s.Charge.Intent.GetEffectiveInvoiceAt(),
	})
	if err != nil {
		return fmt.Errorf("creating gathering line for %s period: %w", input.Op, err)
	}

	// We are in pre-active state, so only the gathering line exists
	if currentRun == nil {
		if input.NewAmountAfterProration.IsZero() {
			// A zero patch target has no invoice artifact to wait for. Keep it
			// terminal and clear advancement so the charge worker stops
			// selecting it.
			s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))
			s.Charge.Status = flatfee.StatusFinal
			s.Charge.State.AdvanceAfter = nil
			return nil
		}

		// Gathering invoices do not have a charge realization run yet, so the
		// invoice artifact is derived entirely from the effective charge intent.
		// Updating by charge ID is enough here: no downstream state points at
		// gathering-line detailed rows, and billing can retain the existing
		// pending line identity.
		s.AddInvoicePatch(invoiceupdater.NewUpsertGatheringLineByChargeIDPatch(s.Charge.ID, updatedGatheringLine))
		// A zero charge can become billable again after extend/shrink. Move it
		// back to created so normal invoice_at advancement and invoicing can
		// recreate the CTI lifecycle.
		s.Charge.Status = flatfee.StatusCreated
		return s.AdvanceAfterInvoiceAt(ctx)
	}

	// Run exists, so we started the billing cycle, thus we don't have a gathering line, but we do have a standard line

	// Let's validate that the run has a persisted line references, before continuing
	if currentRun.LineID == nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot %s flat-fee charge %s because current realization run %s does not have a persisted line reference", input.Op, s.Charge.ID, currentRun.ID.ID),
		)
	}

	if currentRun.InvoiceID == nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("cannot %s flat-fee charge %s because current realization run %s does not have a persisted invoice reference", input.Op, s.Charge.ID, currentRun.ID.ID),
		)
	}

	// A mutable, non-terminal run can update its draft invoice standard line in place.
	if !currentRun.Immutable && !s.IsCurrentRunZeroFiatAmountOverage() {
		// Case #1: If the new amount is zero we just need to delete the old line
		if input.NewAmountAfterProration.IsZero() {
			if err := s.cancelCurrentRealization(ctx); err != nil {
				return fmt.Errorf("canceling zero-amount current run: %w", err)
			}

			// After the line is removed, a zero-amount charge has no remaining
			// invoice lifecycle to wait for.
			s.Charge.Status = flatfee.StatusFinal
			s.Charge.State.AdvanceAfter = nil

			return nil
		}

		line, err := updatedGatheringLine.AsNewStandardLine(*currentRun.InvoiceID)
		if err != nil {
			return fmt.Errorf("converting %s flat-fee gathering line target to standard line: %w", input.Op, err)
		}

		line.ID = *currentRun.LineID

		// The mutable run still describes the previous effective intent.
		// Rerate and reconcile it first, then project the resulting charge state
		// onto the billing-owned line identity.
		reconciledRun, err := s.Realizations.ReconcileRunToIntent(ctx, flatfeerealizations.ReconcileRunToIntentInput{
			Charge: s.Charge,
			Run:    *currentRun,
		})
		if err != nil {
			return fmt.Errorf("reconcile run to intent for %s flat-fee charge[%s]: %w", input.Op, s.Charge.ID, err)
		}

		s.Charge.Realizations.CurrentRun = &reconciledRun
		if err := populateFlatFeeStandardLineFromRun(line, populateFlatFeeStandardLineFromRunInput{
			Charge: s.Charge,
			Run:    reconciledRun,
			Stage:  standardLinePopulationStageIntentReconciliation,
		}); err != nil {
			return fmt.Errorf("mapping reconciled flat-fee run to standard line[%s]: %w", line.ID, err)
		}

		genericLine, err := line.AsInvoiceLine().AsGenericLine()
		if err != nil {
			return fmt.Errorf("converting %s flat-fee standard line[%s] to generic line: %w", input.Op, *currentRun.LineID, err)
		}

		s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(genericLine))
		return nil
	}

	// Final case: externally issued and zero-fiat terminal history cannot be
	// rerated in place, so invoke the replacement path unless the amount is unchanged.
	if input.NewAmountAfterProration.Equal(input.OldAmountAfterProration) {
		return nil
	}

	// We need to trigger a prorating for the new amount

	if err := s.cancelCurrentRealization(ctx); err != nil {
		return fmt.Errorf("canceling current run before replacement: %w", err)
	}

	s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(updatedGatheringLine))

	s.Charge.Status = flatfee.StatusCreated
	return s.AdvanceAfterInvoiceAt(ctx)
}

// correctReversibleCurrentRun removes invoice-finalization effects that would
// otherwise make rerating a mutable custom-currency run reuse stale preparation.
func (s *CreditThenInvoiceStateMachine) correctReversibleCurrentRun(ctx context.Context, op meta.PatchType) error {
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun == nil || currentRun.Immutable || currentRun.AccruedUsage == nil {
		return nil
	}

	if !s.Charge.Intent.GetCurrency().IsCustom() {
		return fmt.Errorf("cannot %s flat-fee charge %s: mutable realization run %s has unexpected regular-fiat invoice usage", op, s.Charge.ID, currentRun.ID.ID)
	}

	_, err := s.Realizations.CorrectPreparedCustomCurrencyInvoiceRealizations(ctx, flatfeerealizations.CreditReconciliationHandlerInput{
		Charge:     s.Charge,
		Run:        *currentRun,
		AllocateAt: flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), currentRun.ServicePeriod),
	})
	if err != nil {
		return fmt.Errorf("correct reversible realization run[%s] before %s: %w", currentRun.ID.ID, op, err)
	}

	if err := s.RefetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch flat-fee charge[%s] after correcting realization run[%s]: %w", s.Charge.ID, currentRun.ID.ID, err)
	}

	return nil
}
