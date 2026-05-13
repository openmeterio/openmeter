package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeFlatFee
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return true, nil
}

func (e *LineEngine) SplitGatheringLine(context.Context, billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("flat fee line is not progressively billed")
}

func (e *LineEngine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		stdLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to standard line: %w", err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return stdLines, nil
}

func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	stdLines, err := slicesx.MapWithErr(input.Lines, func(stdLine *billing.StandardLine) (*billing.StandardLine, error) {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return nil, err
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerFinalInvoiceCreated, billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return nil, fmt.Errorf("triggering %s for charge[%s]: %w", meta.TriggerFinalInvoiceCreated, stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s] after %s: %w", stateMachine.GetCharge().ID, meta.TriggerFinalInvoiceCreated, err)
		}

		charge := stateMachine.GetCharge()
		if charge.Realizations.CurrentRun == nil {
			return nil, fmt.Errorf("flat fee charge[%s]: current run is required for line[%s]", charge.ID, stdLine.ID)
		}

		if err := populateFlatFeeStandardLineFromRun(stdLine, *charge.Realizations.CurrentRun); err != nil {
			return nil, fmt.Errorf("populating standard line from run for charge[%s]: %w", charge.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return stdLines, nil
}

func (e *LineEngine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return nil, err
		}

		canFire, err := stateMachine.CanFire(ctx, meta.TriggerCollectionCompleted)
		if err != nil {
			return nil, fmt.Errorf("checking collection_completed for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			continue
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerCollectionCompleted); err != nil {
			return nil, fmt.Errorf("triggering collection_completed for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing flat fee charge[%s] after collection_completed: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return input.Lines, nil
}

func (e *LineEngine) OnMutableStandardLinesDeleted(ctx context.Context, input billing.OnMutableStandardLinesDeletedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
		meta.ExpandDetailedLines,
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charges for deleted standard lines: %w", err)
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("flat fee charge[%s] not found for deleted standard line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		run, err := charge.Realizations.GetByLineID(stdLine.ID)
		if err != nil {
			return err
		}

		if run.DeletedAt != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
		}

		if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, input.Invoice.ID)
		}

		if run.AccruedUsage != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] has invoice accrued allocation", stdLine.ID, run.ID.ID)
		}

		if run.Payment != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] has payment allocation", stdLine.ID, run.ID.ID)
		}

		if charge.Realizations.CurrentRun != nil && charge.Realizations.CurrentRun.ID.ID == run.ID.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because realization run[%s] is still current for charge[%s]", stdLine.ID, run.ID.ID, charge.ID)
		}

		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator for charge[%s]: %w", charge.ID, err)
		}

		if _, err := e.service.realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             charge,
			Run:                run,
			AllocateAt:         clock.Now(),
			CurrencyCalculator: currencyCalculator,
		}); err != nil {
			return fmt.Errorf("correcting credits for deleted flat fee standard line[%s] run[%s]: %w", stdLine.ID, run.ID.ID, err)
		}

		if err := e.service.adapter.UpsertDetailedLines(ctx, run.ID, nil); err != nil {
			return fmt.Errorf("deleting detailed lines for deleted flat fee standard line[%s] run[%s]: %w", stdLine.ID, run.ID.ID, err)
		}

		if _, err := e.service.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:        run.ID,
			DeletedAt: mo.Some(lo.ToPtr(clock.Now())),
		}); err != nil {
			return fmt.Errorf("marking realization run[%s] deleted for flat fee standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnUnsupportedCreditNote(ctx context.Context, input billing.OnUnsupportedCreditNoteInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLineEvent(ctx, input, meta.Expands{
		meta.ExpandRealizations,
	})
	if err != nil {
		return fmt.Errorf("getting flat fee charges for unsupported credit note: %w", err)
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("flat fee charge[%s] not found for unsupported credit note line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		run, err := charge.Realizations.GetByLineID(stdLine.ID)
		if err != nil {
			return err
		}

		if run.InvoiceID == nil || *run.InvoiceID != input.Invoice.ID {
			return fmt.Errorf("flat fee standard line[%s] cannot be marked unsupported credit note because realization run[%s] is not associated with invoice[%s]", stdLine.ID, run.ID.ID, input.Invoice.ID)
		}

		if run.DeletedAt != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be marked unsupported credit note because realization run[%s] is already deleted", stdLine.ID, run.ID.ID)
		}

		if run.Type == flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote {
			continue
		}

		if _, err := e.service.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:   run.ID,
			Type: mo.Some(flatfee.RealizationRunTypeInvalidDueToUnsupportedCreditNote),
		}); err != nil {
			return fmt.Errorf("marking realization run[%s] invalid due to unsupported credit note for flat fee standard line[%s]: %w", run.ID.ID, stdLine.ID, err)
		}
	}

	return nil
}

func (e *LineEngine) newStateMachineForStandardLine(ctx context.Context, stdLine *billing.StandardLine) (*CreditThenInvoiceStateMachine, error) {
	if stdLine == nil {
		return nil, fmt.Errorf("flat fee standard line is nil")
	}

	if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
		return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
	}

	charge, err := e.service.GetByID(ctx, flatfee.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: stdLine.Namespace,
			ID:        *stdLine.ChargeID,
		},
		Expands: meta.Expands{
			meta.ExpandRealizations,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting flat fee charge for line[%s]: %w", stdLine.ID, err)
	}

	if charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		return nil, fmt.Errorf(
			"flat fee standard line[%s]: unsupported settlement mode for standard invoice lifecycle: %s",
			stdLine.ID,
			charge.Intent.SettlementMode,
		)
	}

	stateMachine, err := e.service.newStateMachine(StateMachineConfig{
		Charge:       charge,
		Adapter:      e.service.adapter,
		Realizations: e.service.realizations,
		Service:      e.service,
	})
	if err != nil {
		return nil, fmt.Errorf("new state machine for flat fee charge[%s]: %w", charge.ID, err)
	}

	creditThenInvoiceStateMachine, ok := stateMachine.(*CreditThenInvoiceStateMachine)
	if !ok {
		return nil, fmt.Errorf("BUG: flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
	}

	return creditThenInvoiceStateMachine, nil
}

func (e *LineEngine) getChargesForStandardLineEvent(ctx context.Context, input billing.StandardLineEventInput, expands meta.Expands) (map[string]flatfee.Charge, error) {
	chargeIDs := make([]string, 0, len(input.Lines))
	seenChargeIDs := make(map[string]struct{}, len(input.Lines))

	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		if stdLine.Namespace != input.Invoice.Namespace {
			return nil, fmt.Errorf("flat fee standard line[%s]: namespace %s does not match invoice namespace %s", stdLine.ID, stdLine.Namespace, input.Invoice.Namespace)
		}

		if _, ok := seenChargeIDs[*stdLine.ChargeID]; ok {
			continue
		}

		seenChargeIDs[*stdLine.ChargeID] = struct{}{}
		chargeIDs = append(chargeIDs, *stdLine.ChargeID)
	}

	charges, err := e.service.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: input.Invoice.Namespace,
		IDs:       chargeIDs,
		Expands:   expands,
	})
	if err != nil {
		return nil, fmt.Errorf("getting flat fee charges: %w", err)
	}

	return lo.KeyBy(charges, func(charge flatfee.Charge) string {
		return charge.ID
	}), nil
}

func (e *LineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerInvoiceIssued, billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("triggering invoice_issued for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return fmt.Errorf("advancing flat fee charge[%s] after invoice_issued: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := e.service.postInvoicePaymentAuthorized(ctx, stateMachine.GetCharge(), billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("authorizing invoice payment for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}

func (e *LineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		if err := e.service.postInvoicePaymentSettled(ctx, stateMachine.GetCharge(), billing.StandardLineWithInvoiceHeader{
			Line:    stdLine,
			Invoice: input.Invoice,
		}); err != nil {
			return fmt.Errorf("settling invoice payment for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := stateMachine.RefetchCharge(ctx); err != nil {
			return fmt.Errorf("refetching flat fee charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		canFire, err := stateMachine.CanFire(ctx, meta.TriggerAllPaymentsSettled)
		if err != nil {
			return fmt.Errorf("checking all_payments_settled for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			continue
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerAllPaymentsSettled); err != nil {
			return fmt.Errorf("triggering all_payments_settled for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return nil
}
