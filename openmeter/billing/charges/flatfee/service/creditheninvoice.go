package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
		// line to produce TriggerFinalInvoiceCreated, so the charge closes
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
		Permit(meta.TriggerAttachInvoiceLine, flatfee.StatusActiveRealizationProcessing).
		OnActive(s.AdvanceAfterInvoiceAt)

	s.Configure(flatfee.StatusActive).
		// This also repairs previously active zero-amount charges. They have
		// no line-engine path left, so active must not become their terminal
		// operational state.
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.IsZeroAmount)).
		Permit(meta.TriggerFinalInvoiceCreated, flatfee.StatusActiveRealizationStarted).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		OnActive(s.AdvanceAfterServicePeriodTo)

	s.Configure(flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationWaitingForCollection).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		OnEntryFrom(meta.TriggerFinalInvoiceCreated, statelessx.WithParameters(s.StartRealization))

	s.Configure(flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerCollectionCompleted, flatfee.StatusActiveRealizationProcessing).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit))

	s.Configure(flatfee.StatusActiveRealizationProcessing).
		Permit(meta.TriggerInvoiceIssued, flatfee.StatusActiveRealizationIssuing).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		OnEntryFrom(meta.TriggerAttachInvoiceLine, statelessx.WithParameters(s.AttachInvoiceLine))

	s.Configure(flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationCompleted).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.AccrueInvoiceUsage))

	s.Configure(flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerNext, flatfee.StatusActiveAwaitingPaymentSettlement).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation))

	s.Configure(flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		Permit(meta.TriggerAllPaymentsSettled, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit))

	s.Configure(flatfee.StatusFinal).
		InternalTransition(meta.TriggerDelete, statelessx.WithParameters(s.DeleteCharge)).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.LineManualEdit)).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		InternalTransition(meta.TriggerLineManualEdit, statelessx.WithParameters(s.UnsupportedLineManualEditOperation))
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, patch meta.PatchDelete) error {
	deletedAt := lo.ToPtr(clock.Now())
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
	}

	if err := s.mutateIntentLayer(ctx, target, func(fields *flatfee.IntentMutableFields) {
		fields.IntentDeletedAt = deletedAt
	}); err != nil {
		return fmt.Errorf("deleting intent: %w", err)
	}

	if target == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// the customer-facing charge and invoice history remain owned by the override.
		return nil
	}

	s.Charge.Status = flatfee.StatusDeleted

	patches := []invoiceupdater.Patch{
		invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID),
	}
	currentRun := s.Charge.Realizations.CurrentRun
	if currentRun != nil && currentRun.LineID != nil && currentRun.InvoiceID != nil {
		patches = append(patches, invoiceupdater.NewDeleteLinePatch(
			billing.LineID{
				Namespace: s.Charge.Namespace,
				ID:        *currentRun.LineID,
			},
			*currentRun.InvoiceID,
		))

		if err := s.Adapter.DetachCurrentRun(ctx, s.Charge.GetChargeID()); err != nil {
			return fmt.Errorf("detach current run before deleting charge: %w", err)
		}

		s.Charge.Realizations.PriorRuns = append(s.Charge.Realizations.PriorRuns, *currentRun)
		s.Charge.Realizations.CurrentRun = nil
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
	invoicingStateInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !invoicingStateInput.ShouldReconcile {
		return nil
	}

	return s.reconcileInvoicingState(ctx, invoicingStateInput)
}

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	invoicingStateInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	if !invoicingStateInput.ShouldReconcile {
		return nil
	}

	return s.reconcileInvoicingState(ctx, invoicingStateInput)
}

func (s *CreditThenInvoiceStateMachine) LineManualEdit(ctx context.Context, patch meta.PatchLineManualEdit) error {
	target, err := patch.GetTargetLayer(s.Charge.Intent)
	if err != nil {
		return fmt.Errorf("getting patch target layer: %w", err)
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
		ShouldReconcile:         true,
		Op:                      meta.PatchTypeLineManualEdit,
		Period:                  s.Charge.Intent.GetEffectiveServicePeriod(),
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

	if target == meta.ChangeTargetBase && s.Charge.Intent.HasOverrideLayer() {
		// Subscription sync targets the base intent. When an override is active,
		// the customer-facing invoice remains owned by the override layer.
		return reconcileInvoicingStateInput{}, nil
	}

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return reconcileInvoicingStateInput{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	return reconcileInvoicingStateInput{
		ShouldReconcile:         true,
		Op:                      patch.Op(),
		Period:                  intent.GetEffectiveServicePeriod(),
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

// StartRealization creates the current run. The line engine maps the run back
// onto the returned standard line before billing persists line updates.
func (s *CreditThenInvoiceStateMachine) StartRealization(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	result, err := s.Realizations.StartCreditThenInvoiceRun(ctx, flatfeerealizations.StartCreditThenInvoiceRunInput{
		Charge:  s.Charge,
		Line:    *input.Line,
		Invoice: input.Invoice,
	})
	if err != nil {
		return fmt.Errorf("start credit-then-invoice run: %w", err)
	}

	s.Charge.Realizations.CurrentRun = &result.Run

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
		Charge:        s.Charge,
		ServicePeriod: s.Charge.Intent.GetEffectiveServicePeriod(),
		InvoiceAt:     s.Charge.Intent.GetEffectiveInvoiceAt(),
	})
	if err != nil {
		return fmt.Errorf("creating flat-fee attach target line: %w", err)
	}

	line, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
	if err != nil {
		return fmt.Errorf("converting flat-fee attach target to standard line: %w", err)
	}

	line.ID = input.Line.ID

	result, err := s.Realizations.StartCreditThenInvoiceRun(ctx, flatfeerealizations.StartCreditThenInvoiceRunInput{
		Charge:  s.Charge,
		Line:    *line,
		Invoice: input.Invoice,
	})
	if err != nil {
		return fmt.Errorf("start attached credit-then-invoice run: %w", err)
	}

	s.Charge.Realizations.CurrentRun = &result.Run

	if err := populateFlatFeeStandardLineFromRun(line, result.Run); err != nil {
		return fmt.Errorf("mapping attached flat-fee run to standard line[%s]: %w", line.ID, err)
	}

	s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(line.AsGenericLine()))

	return nil
}

func (s *CreditThenInvoiceStateMachine) AccrueInvoiceUsage(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	result, err := s.Realizations.AccrueInvoiceUsage(ctx, flatfeerealizations.AccrueInvoiceUsageInput{
		Charge:         s.Charge,
		LineWithHeader: input,
	})
	if err != nil {
		return fmt.Errorf("post invoice issued: %w", err)
	}

	// The state machine persists this clear through StatusFinal's ClearAdvanceAfter hook.
	s.Charge.Realizations.CurrentRun = &result.Run
	s.Charge.State.AdvanceAfter = nil

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

type reconcileInvoicingStateInput struct {
	ShouldReconcile         bool
	Op                      meta.PatchType
	Period                  timeutil.ClosedPeriod
	Intent                  flatfee.OverridableIntent
	OldAmountAfterProration alpacadecimal.Decimal
	NewAmountAfterProration alpacadecimal.Decimal
}

func (s *CreditThenInvoiceStateMachine) reconcileInvoicingState(ctx context.Context, input reconcileInvoicingStateInput) error {
	currentRun := s.Charge.Realizations.CurrentRun

	// TODO(credit-note support): this branch is a temporary fallback for
	// immutable invoice lines until the line updater can correct them with
	// credit notes. The normal patch flow below assumes immutable invoice
	// history can be adjusted safely; while that is false, we update the
	// charge intent/state but avoid creating replacement billable work for
	// the already-invoiced period.
	if !s.CreditNotesSupported {
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
		Charge:        s.Charge,
		ServicePeriod: input.Period,
		InvoiceAt:     s.Charge.Intent.GetEffectiveInvoiceAt(),
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
		// back to created so normal service-period advancement and invoicing
		// can recreate the CTI lifecycle.
		s.Charge.Status = flatfee.StatusCreated
		s.Charge.State.AdvanceAfter = lo.ToPtr(meta.NormalizeTimestamp(input.Period.From))
		return nil
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

	// If the run is not immutable, we can just update the invoice standard line.
	if !currentRun.Immutable {
		// Case #1: If the new amount is zero we just need to delete the old line
		if input.NewAmountAfterProration.IsZero() {
			s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
				billing.LineID{
					Namespace: s.Charge.Namespace,
					ID:        *currentRun.LineID,
				},
				*currentRun.InvoiceID,
			))

			if err := s.Adapter.DetachCurrentRun(ctx, s.Charge.GetChargeID()); err != nil {
				return fmt.Errorf("detach zero-amount current run: %w", err)
			}

			s.Charge.Realizations.PriorRuns = append(s.Charge.Realizations.PriorRuns, *currentRun)
			s.Charge.Realizations.CurrentRun = nil

			// The mutable standard-line deletion hook owns credit correction
			// for the detached run. After the line is removed, a zero-amount
			// charge has no remaining invoice lifecycle to wait for.
			s.Charge.Status = flatfee.StatusFinal
			s.Charge.State.AdvanceAfter = nil

			return nil
		}

		line, err := updatedGatheringLine.AsNewStandardLine(*currentRun.InvoiceID)
		if err != nil {
			return fmt.Errorf("converting %s flat-fee gathering line target to standard line: %w", input.Op, err)
		}

		line.ID = *currentRun.LineID

		// The invoice updater rebuilt the mutable standard line from the new
		// charge intent, but the charge realization run still describes the old
		// line amount and credit allocations. Reconcile them before handing the
		// updated line back to billing.
		result, err := s.Realizations.ReconcileStandardLineToIntent(ctx, flatfeerealizations.ReconcileStandardLineToIntentInput{
			Charge:     s.Charge,
			Run:        *currentRun,
			Line:       *line,
			AllocateAt: flatfee.UsageBookedAt(s.Charge.Intent.GetEffectivePaymentTerm(), currentRun.ServicePeriod),
		})
		if err != nil {
			return fmt.Errorf("reconcile standard line to intent for %s flat-fee charge[%s]: %w", input.Op, s.Charge.ID, err)
		}

		s.Charge.Realizations.CurrentRun = &result.Run
		line = &result.Line

		genericLine, err := line.AsInvoiceLine().AsGenericLine()
		if err != nil {
			return fmt.Errorf("converting %s flat-fee standard line[%s] to generic line: %w", input.Op, *currentRun.LineID, err)
		}

		s.AddInvoicePatch(invoiceupdater.NewUpdateLinePatch(genericLine))
		return nil
	}

	// Final case: we have an immutable invoice, so we need to invoke the prorating path, unless the amount haven't changed
	if input.NewAmountAfterProration.Equal(input.OldAmountAfterProration) {
		return nil
	}

	// We need to trigger a prorating for the new amount

	s.AddInvoicePatch(invoiceupdater.NewDeleteLinePatch(
		billing.LineID{
			Namespace: s.Charge.Namespace,
			ID:        *currentRun.LineID,
		},
		*currentRun.InvoiceID,
	))

	if err := s.Adapter.DetachCurrentRun(ctx, s.Charge.GetChargeID()); err != nil {
		return fmt.Errorf("detach immutable current run: %w", err)
	}

	s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(updatedGatheringLine))

	s.Charge.Realizations.PriorRuns = append(s.Charge.Realizations.PriorRuns, *currentRun)
	s.Charge.Realizations.CurrentRun = nil

	s.Charge.Status = flatfee.StatusCreated
	advanceAfter := meta.NormalizeTimestamp(input.Period.From)
	s.Charge.State.AdvanceAfter = &advanceAfter

	return nil
}
