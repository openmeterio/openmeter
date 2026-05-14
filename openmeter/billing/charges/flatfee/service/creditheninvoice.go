package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

type generateInvoicePatchesInput struct {
	Op                      meta.PatchType
	Period                  timeutil.ClosedPeriod
	OldAmountAfterProration alpacadecimal.Decimal
	NewAmountAfterProration alpacadecimal.Decimal
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

	out := &CreditThenInvoiceStateMachine{
		stateMachine: stateMachine,
	}
	out.configureStates()

	return out, nil
}

func (s *CreditThenInvoiceStateMachine) configureStates() {
	s.Configure(flatfee.StatusCreated).
		Permit(
			meta.TriggerNext,
			flatfee.StatusActive,
			statelessx.BoolFn(s.IsInsideServicePeriod),
		).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		InternalTransition(meta.TriggerExtend, statelessx.WithParameters(s.ExtendCharge)).
		InternalTransition(meta.TriggerShrink, statelessx.WithParameters(s.ShrinkCharge)).
		OnActive(s.AdvanceAfterServicePeriodFrom)

	s.Configure(flatfee.StatusActive).
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

	s.Charge.Intent.ServicePeriod.To = patch.GetNewServicePeriodTo()
	s.Charge.Intent.FullServicePeriod.To = patch.GetNewFullServicePeriodTo()
	s.Charge.Intent.BillingPeriod.To = patch.GetNewBillingPeriodTo()
	s.Charge.Intent.InvoiceAt = patch.GetNewInvoiceAt()
	s.Charge.Intent = s.Charge.Intent.Normalized()

	amountAfterProration, err := s.Charge.Intent.CalculateAmountAfterProration()
	if err != nil {
		return generateInvoicePatchesInput{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	s.Charge.State.AmountAfterProration = amountAfterProration

	return generateInvoicePatchesInput{
		Op:                      patch.Op(),
		Period:                  s.Charge.Intent.ServicePeriod,
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

func (s *CreditThenInvoiceStateMachine) generateInvoicePatches(ctx context.Context, input generateInvoicePatchesInput) error {
	currentRun := s.Charge.Realizations.CurrentRun

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
		s.AddInvoicePatch(invoiceupdater.NewCreateLinePatch(updatedGatheringLine))
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
		line, err := updatedGatheringLine.AsNewStandardLine(*currentRun.InvoiceID)
		if err != nil {
			return fmt.Errorf("converting %s flat-fee gathering line target to standard line: %w", input.Op, err)
		}

		line.ID = *currentRun.LineID

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
