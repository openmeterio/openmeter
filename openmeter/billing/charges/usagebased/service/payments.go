package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrun "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/run"
)

type recordRunPaymentsInput struct {
	Lines    billing.StandardLines
	Invoice  billing.StandardInvoice
	RecordFn func(ctx context.Context, stateMachine StateMachine, line billing.StandardLine, invoice billing.StandardInvoice) error
}

func (i recordRunPaymentsInput) Validate() error {
	if len(i.Lines) == 0 {
		return fmt.Errorf("lines are required")
	}

	if i.Invoice.ID == "" {
		return fmt.Errorf("invoice is required")
	}

	if i.RecordFn == nil {
		return fmt.Errorf("recordFn is required")
	}

	return nil
}

func (e *LineEngine) recordRunPayments(ctx context.Context, input recordRunPaymentsInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating record run payments input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := input.RecordFn(ctx, stateMachine, *stdLine, input.Invoice); err != nil {
			return err
		}
	}

	return nil
}

func (e *LineEngine) recordPaymentAuthorized(ctx context.Context, stateMachine StateMachine, line billing.StandardLine, invoice billing.StandardInvoice) error {
	charge := stateMachine.GetCharge()

	run, err := getRunForLine(charge, line.ID)
	if err != nil {
		return err
	}

	_, err = e.service.runs.BookInvoicedPaymentAuthorized(ctx, usagebasedrun.BookInvoicedPaymentAuthorizedInput{
		Charge:  charge,
		Run:     run,
		Invoice: invoice,
		Line:    line,
	})
	if err != nil {
		return fmt.Errorf("authorize invoice payment for charge[%s]: %w", charge.ID, err)
	}

	return nil
}

func (e *LineEngine) recordPaymentSettled(ctx context.Context, stateMachine StateMachine, line billing.StandardLine, invoice billing.StandardInvoice) error {
	charge := stateMachine.GetCharge()

	run, err := getRunForLine(charge, line.ID)
	if err != nil {
		return err
	}

	result, err := e.service.runs.SettleInvoicedPayment(ctx, usagebasedrun.SettleInvoicedPaymentInput{
		Charge:  charge,
		Run:     run,
		Invoice: invoice,
		Line:    line,
	})
	if err != nil {
		return fmt.Errorf("settle invoice payment for charge[%s]: %w", charge.ID, err)
	}

	if err := charge.Realizations.SetRealizationRun(result.Run); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	if charge.Status != usagebased.StatusActiveAwaitingPaymentSettlement {
		return nil
	}

	if !areAllInvoicedRunsSettled(charge) {
		return nil
	}

	stateMachineConfig, err := e.service.getStateMachineConfigForPatch(ctx, charge)
	if err != nil {
		return fmt.Errorf("get state machine config: %w", err)
	}

	settlementStateMachine, err := e.service.newStateMachine(stateMachineConfig)
	if err != nil {
		return fmt.Errorf("new state machine: %w", err)
	}

	if err := settlementStateMachine.FireAndActivate(ctx, meta.TriggerAllPaymentsSettled); err != nil {
		return fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerAllPaymentsSettled, charge.ID, err)
	}

	return nil
}

func areAllInvoicedRunsSettled(charge usagebased.Charge) bool {
	hasFinalInvoicedRun := false

	for _, run := range charge.Realizations {
		if run.Type == usagebased.RealizationRunTypeFinalRealization && run.InvoiceUsage != nil {
			hasFinalInvoicedRun = true
		}

		if run.InvoiceUsage == nil {
			continue
		}

		if run.Payment == nil || run.Payment.Status != payment.StatusSettled {
			return false
		}
	}

	return hasFinalInvoicedRun
}
