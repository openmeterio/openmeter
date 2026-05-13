package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
		OnActive(s.AdvanceAfterServicePeriodFrom)

	s.Configure(flatfee.StatusActive).
		Permit(meta.TriggerFinalInvoiceCreated, flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnActive(s.AdvanceAfterServicePeriodTo)

	s.Configure(flatfee.StatusActiveRealizationStarted).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnEntryFrom(meta.TriggerFinalInvoiceCreated, statelessx.WithParameters(s.StartRealization))

	s.Configure(flatfee.StatusActiveRealizationWaitingForCollection).
		Permit(meta.TriggerCollectionCompleted, flatfee.StatusActiveRealizationProcessing).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted)

	s.Configure(flatfee.StatusActiveRealizationProcessing).
		Permit(meta.TriggerInvoiceIssued, flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted)

	s.Configure(flatfee.StatusActiveRealizationIssuing).
		Permit(meta.TriggerNext, flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnEntryFrom(meta.TriggerInvoiceIssued, statelessx.WithParameters(s.AccrueInvoiceUsage))

	s.Configure(flatfee.StatusActiveRealizationCompleted).
		Permit(meta.TriggerNext, flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted)

	s.Configure(flatfee.StatusActiveAwaitingPaymentSettlement).
		Permit(meta.TriggerNext, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		Permit(meta.TriggerAllPaymentsSettled, flatfee.StatusFinal, statelessx.BoolFn(s.AreAllPaymentsSettled)).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted)

	s.Configure(flatfee.StatusFinal).
		Permit(meta.TriggerDelete, flatfee.StatusDeleted).
		OnActive(s.ClearAdvanceAfter)

	s.Configure(flatfee.StatusDeleted).
		OnEntry(statelessx.WithParameters(s.DeleteCharge))
}

func (s *CreditThenInvoiceStateMachine) DeleteCharge(ctx context.Context, _ meta.PatchDeletePolicy) error {
	patches := []invoiceupdater.Patch{
		invoiceupdater.NewDeleteGatheringLineByChargeIDPatch(s.Charge.ID),
	}

	if s.Charge.Realizations.CurrentRun != nil &&
		s.Charge.Realizations.CurrentRun.LineID != nil &&
		s.Charge.Realizations.CurrentRun.InvoiceID != nil {
		patches = append(patches, invoiceupdater.NewDeleteLinePatch(
			billing.LineID{
				Namespace: s.Charge.Namespace,
				ID:        *s.Charge.Realizations.CurrentRun.LineID,
			},
			*s.Charge.Realizations.CurrentRun.InvoiceID,
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

// StartRealization mutates input.Line.CreditsApplied after allocating credits.
// The line engine relies on that in-place update when persisting detailed lines.
func (s *CreditThenInvoiceStateMachine) StartRealization(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if s.Charge.Realizations.CurrentRun == nil {
		runBase, err := s.Adapter.ProvisionCurrentRun(ctx, flatfee.ProvisionCurrentRunInput{
			Charge:                    s.Charge.ChargeBase,
			NoFiatTransactionRequired: s.Charge.State.AmountAfterProration.IsZero(),
		})
		if err != nil {
			return fmt.Errorf("provision current run: %w", err)
		}

		s.Charge.Realizations.CurrentRun = &flatfee.RealizationRun{
			RealizationRunBase: runBase,
		}
	}

	runBase, err := s.Adapter.AssignCurrentRunInvoiceLine(ctx, flatfee.AssignCurrentRunInvoiceLineInput{
		ChargeID:  s.Charge.GetChargeID(),
		LineID:    input.Line.ID,
		InvoiceID: input.Invoice.ID,
	})
	if err != nil {
		return fmt.Errorf("assign invoice line to current run: %w", err)
	}

	s.Charge.Realizations.CurrentRun.RealizationRunBase = runBase

	realizations, err := s.Service.postLineAssignedToInvoice(ctx, s.Charge, *input.Line)
	if err != nil {
		return fmt.Errorf("assign line to invoice: %w", err)
	}

	if len(realizations) > 0 {
		input.Line.CreditsApplied = convertCreditRealizations(realizations)
	}

	s.Charge.Realizations.CurrentRun.CreditRealizations = append(s.Charge.Realizations.CurrentRun.CreditRealizations, realizations...)

	return nil
}

func (s *CreditThenInvoiceStateMachine) AccrueInvoiceUsage(ctx context.Context, input billing.StandardLineWithInvoiceHeader) error {
	if err := input.Validate(); err != nil {
		return err
	}

	accruedUsage, err := s.Service.accrueInvoiceUsage(ctx, s.Charge, input)
	if err != nil {
		return fmt.Errorf("post invoice issued: %w", err)
	}

	s.Charge.Realizations.CurrentRun.AccruedUsage = accruedUsage

	// The state machine persists this clear through StatusFinal's ClearAdvanceAfter hook.
	s.Charge.State.AdvanceAfter = nil

	return nil
}

func (s *CreditThenInvoiceStateMachine) AreAllPaymentsSettled() bool {
	if s.Charge.Realizations.CurrentRun == nil {
		return false
	}

	if s.Charge.Realizations.CurrentRun.AccruedUsage == nil {
		return true
	}

	if s.Charge.Realizations.CurrentRun.Payment == nil {
		return false
	}

	return s.Charge.Realizations.CurrentRun.Payment.Status == payment.StatusSettled
}
