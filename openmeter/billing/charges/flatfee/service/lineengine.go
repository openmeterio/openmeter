package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeerealizations "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service/realizations"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var (
	_ billing.LineEngine     = (*LineEngine)(nil)
	_ billing.LineCalculator = (*LineEngine)(nil)
)

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

	return e.CalculateLines(billing.CalculateLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	})
}

func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	chargesByID := make(map[string]flatfee.Charge, len(input.Lines))
	for _, stdLine := range input.Lines {
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
		chargesByID[charge.ID] = charge
	}

	lines, err := e.CalculateLines(billing.CalculateLinesInput(input))
	if err != nil {
		return nil, err
	}

	for _, stdLine := range lines {
		if stdLine.ChargeID == nil {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return nil, fmt.Errorf("flat fee charge not found for line[%s]: %s", stdLine.ID, *stdLine.ChargeID)
		}

		if err := e.service.persistDetailedLines(ctx, charge, *stdLine); err != nil {
			return nil, fmt.Errorf("persisting detailed lines for line[%s]: %w", stdLine.ID, err)
		}
	}

	return lines, nil
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

	chargesByID, err := e.getChargesForMutableStandardLineDelete(ctx, input)
	if err != nil {
		return err
	}

	cleanedChargeIDs := make(map[string]struct{}, len(chargesByID))
	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("flat fee charge[%s] not found for deleted standard line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		if _, ok := cleanedChargeIDs[charge.ID]; ok {
			continue
		}

		if charge.Realizations.CurrentRun != nil && charge.Realizations.CurrentRun.AccruedUsage != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because charge[%s] has invoice accrued allocation", stdLine.ID, charge.ID)
		}

		if charge.Realizations.CurrentRun != nil && charge.Realizations.CurrentRun.Payment != nil {
			return fmt.Errorf("flat fee standard line[%s] cannot be deleted because charge[%s] has payment allocation", stdLine.ID, charge.ID)
		}

		currencyCalculator, err := charge.Intent.Currency.Calculator()
		if err != nil {
			return fmt.Errorf("getting currency calculator for charge[%s]: %w", charge.ID, err)
		}

		if _, err := e.service.realizations.CorrectAllCredits(ctx, flatfeerealizations.CorrectAllCreditRealizationsInput{
			Charge:             charge,
			AllocateAt:         clock.Now(),
			CurrencyCalculator: currencyCalculator,
		}); err != nil {
			return fmt.Errorf("correcting credits for deleted flat fee standard line[%s] charge[%s]: %w", stdLine.ID, charge.ID, err)
		}

		if err := e.service.adapter.UpsertDetailedLines(ctx, charge.GetChargeID(), nil); err != nil {
			return fmt.Errorf("deleting detailed lines for deleted flat fee standard line[%s] charge[%s]: %w", stdLine.ID, charge.ID, err)
		}

		cleanedChargeIDs[charge.ID] = struct{}{}
	}

	return nil
}

func (e *LineEngine) OnUnsupportedCreditNote(_ context.Context, _ billing.OnUnsupportedCreditNoteInput) error {
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
		return nil, fmt.Errorf("flat fee charge[%s]: expected credit_then_invoice state machine, got %T", charge.ID, stateMachine)
	}

	return creditThenInvoiceStateMachine, nil
}

func (e *LineEngine) getChargesForMutableStandardLineDelete(ctx context.Context, input billing.OnMutableStandardLinesDeletedInput) (map[string]flatfee.Charge, error) {
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
		Expands: meta.Expands{
			meta.ExpandRealizations,
			meta.ExpandDetailedLines,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting flat fee charges for deleted standard lines: %w", err)
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

func (e *LineEngine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("lines are required")
	}

	for _, stdLine := range input.Lines {
		generatedDetailedLines, err := e.service.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging generated detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}
