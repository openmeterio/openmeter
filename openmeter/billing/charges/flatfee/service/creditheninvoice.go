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
	GetNewServicePeriodTo() time.Time
	GetNewFullServicePeriodTo() time.Time
	GetNewBillingPeriodTo() time.Time
	GetNewInvoiceAt() time.Time
}

var (
	_ periodPatch = meta.PatchExtend{}
	_ periodPatch = meta.PatchShrink{}
)

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

	out := &CreditThenInvoiceStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *CreditThenInvoiceStateMachine) configureStates() {
	s.Configure(flatfee.StatusCreated).
		// Zero-amount CTI flat fees intentionally skip the billing line
		// engine. Once the service period starts there will be no gathering
		// line to produce TriggerFinalInvoiceCreated, so the charge closes
		// directly from created.
		Permit(
			meta.TriggerNext,
			flatfee.StatusFinal,
			statelessx.BoolFn(s.IsInsideServicePeriodAndZeroAmount),
		).
		// Non-zero CTI flat fees still wait in active for the flat-fee line
		// engine to create a realization run from the standard invoice line.
		Permit(
			meta.TriggerNext,
			flatfee.StatusActive,
			statelessx.BoolFn(s.IsInsideServicePeriodAndNonZeroAmount),
		).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterServicePeriodFrom)

	s.Configure(flatfee.StatusActive).
		// This also repairs previously active zero-amount charges. They have
		// no line-engine path left, so active must not become their terminal
		// operational state.
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.IsZeroAmount)).
		Permit(meta.TriggerFinalInvoiceCreated, flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterServicePeriodTo)

	s.Configure(flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnEntryFrom(meta.TriggerFinalInvoiceCreated, statelessx.WithParameters(s.StartRealization))

	s.Configure(flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerCollectionCompleted, flatfee.StatusActiveRealizationProcessing).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(flatfee.StatusActiveRealizationProcessing).
		Permit(meta.TriggerInvoiceIssued, flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.AccrueInvoiceUsage))

	s.Configure(flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerNext, flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation))

	s.Configure(flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		Permit(meta.TriggerAllPaymentsSettled, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge))

	s.Configure(flatfee.StatusFinal).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.UnsupportedExtendOperation)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.UnsupportedShrinkOperation)).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, _ meta.PatchDeletePolicy) error {
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
	if err := patch.ValidateWith(s.Charge.Intent.Intent); err != nil {
		return fmt.Errorf("validate extend patch: %w", err)
	}

	invoicePatchInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	return s.generateInvoicePatches(ctx, invoicePatchInput)
}

func (s *CreditThenInvoiceStateMachine) ShrinkCharge(ctx context.Context, patch meta.PatchShrink) error {
	if err := patch.ValidateWith(s.Charge.Intent.Intent); err != nil {
		return fmt.Errorf("validate shrink patch: %w", err)
	}

	invoicePatchInput, err := s.applyPeriodPatch(patch)
	if err != nil {
		return err
	}

	return s.generateInvoicePatches(ctx, invoicePatchInput)
}

func (s *CreditThenInvoiceStateMachine) applyPeriodPatch(patch periodPatch) (generateInvoicePatchesInput, error) {
	oldAmountAfterProration := s.Charge.State.AmountAfterProration

	intent := s.Charge.Intent
	intent.ServicePeriod.To = patch.GetNewServicePeriodTo()
	intent.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
	intent.BillingPeriod.To = patch.GetNewBillingPeriodTo()
	intent.InvoiceAt = patch.GetNewInvoiceAt()
	intent = intent.Normalized()

	amountAfterProration, err := intent.CalculateAmountAfterProration()
	if err != nil {
		return generateInvoicePatchesInput{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	return generateInvoicePatchesInput{
		Op:                      patch.Op(),
		Period:                  intent.ServicePeriod,
		Intent:                  intent,
		OldAmountAfterProration: oldAmountAfterProration,
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

type generateInvoicePatchesInput struct {
	Op                      meta.PatchType
	Period                  timeutil.ClosedPeriod
	Intent                  flatfee.Intent
	OldAmountAfterProration alpacadecimal.Decimal
	NewAmountAfterProration alpacadecimal.Decimal
}

func (s *CreditThenInvoiceStateMachine) generateInvoicePatches(ctx context.Context, input generateInvoicePatchesInput) error {
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
		InvoiceAt:     s.Charge.Intent.InvoiceAt,
	})
	if err != nil {
		return fmt.Errorf("creating gathering line for %s period: %w", input.Op, err)
	}

	// We are in pre-active state, so only the gathering line exists
	if currentRun == nil {
		s.AddInvoicePatch(invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID))
		if input.NewAmountAfterProration.IsZero() {
			// A zero patch target has no invoice artifact to wait for. Keep it
			// terminal and clear advancement so the charge worker stops
			// selecting it.
			s.Charge.Status = flatfee.StatusFinal
			s.Charge.State.AdvanceAfter = nil
			return nil
		}
		s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(updatedGatheringLine))
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
			AllocateAt: clock.Now(),
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
