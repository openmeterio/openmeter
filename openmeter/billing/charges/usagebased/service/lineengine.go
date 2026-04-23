package service

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeUsageBased
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return !input.AsOf.Before(input.ResolvedBillablePeriod.To), nil
}

func (e *LineEngine) SplitGatheringLine(_ context.Context, input billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	res := billing.SplitGatheringLineResult{}

	if err := input.Validate(); err != nil {
		return res, fmt.Errorf("validating input: %w", err)
	}

	line := input.Line
	if line.ChargeID == nil || *line.ChargeID == "" {
		return res, fmt.Errorf("usage based gathering line[%s]: charge id is required", line.ID)
	}

	if !line.ServicePeriod.Contains(input.SplitAt) {
		return res, fmt.Errorf("usage based gathering line[%s]: splitAt is not within the line period", line.ID)
	}

	postSplitAtLine, err := line.CloneForCreate(func(l *billing.GatheringLine) {
		l.ServicePeriod.From = input.SplitAt
		l.ChildUniqueReferenceID = nil
	})
	if err != nil {
		return res, fmt.Errorf("cloning post split line: %w", err)
	}

	postSplitAtLineEmpty, err := isUsageBasedSplitPeriodEmpty(postSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if post split line is empty: %w", err)
	}

	if !postSplitAtLineEmpty {
		if err := postSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating post split line: %w", err)
		}
	}

	line.ServicePeriod.To = input.SplitAt
	line.InvoiceAt = input.SplitAt
	line.ChildUniqueReferenceID = nil

	preSplitAtLine := line

	preSplitAtLineEmpty, err := isUsageBasedSplitPeriodEmpty(preSplitAtLine)
	if err != nil {
		return res, fmt.Errorf("checking if pre split line is empty: %w", err)
	}

	if preSplitAtLineEmpty {
		preSplitAtLine.DeletedAt = lo.ToPtr(clock.Now())
	} else {
		if err := preSplitAtLine.Validate(); err != nil {
			return res, fmt.Errorf("validating pre split line: %w", err)
		}
	}

	var postSplitAtLinePtr *billing.GatheringLine
	if !postSplitAtLineEmpty {
		postSplitAtLinePtr = &postSplitAtLine
	}

	return billing.SplitGatheringLineResult{
		PreSplitAtLine:  preSplitAtLine,
		PostSplitAtLine: postSplitAtLinePtr,
	}, nil
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

		if stateMachine.GetCharge().Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
			return nil, fmt.Errorf(
				"usage based standard line[%s]: unsupported settlement mode for standard invoice creation: %s",
				stdLine.ID,
				stateMachine.GetCharge().Intent.SettlementMode,
			)
		}

		// Becoming active after the service period starts is not an invoice lifecycle event, so we
		// still rely on the generic TriggerNext/AdvanceUntilStateStable flow before invoice-created
		// lifecycle transitions take over.
		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		trigger := resolveInvoiceCreatedTrigger(stateMachine.GetCharge(), stdLine.Period)
		if stateMachine.GetCharge().State.CurrentRealizationRunID != nil {
			return nil, billing.ValidationError{
				Err: fmt.Errorf("line[%s]: %w", stdLine.ID, usagebased.ErrActiveRealizationRunAlreadyExists),
			}
		}

		if trigger == meta.TriggerPartialInvoiceCreated {
			// A past Period.To intentionally means "collect now" so late events can still produce an immediately collectable partial invoice.
			stdLine.OverrideCollectionPeriodEnd = lo.ToPtr(stdLine.Period.To.Add(usagebased.InternalCollectionPeriod))
		}

		if err := stateMachine.FireAndActivate(ctx, trigger, invoiceCreatedInput{
			LineID:                      stdLine.ID,
			OverrideCollectionPeriodEnd: stdLine.OverrideCollectionPeriodEnd,
		}); err != nil {
			return nil, fmt.Errorf("triggering %s for charge[%s]: %w", trigger, stateMachine.GetCharge().ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s] after %s: %w", stateMachine.GetCharge().ID, trigger, err)
		}

		currentRun, err := stateMachine.GetCharge().GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := populateUsageBasedStandardLineFromRun(stdLine, currentRun); err != nil {
			return nil, fmt.Errorf("populating standard line from run for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
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
			return nil, fmt.Errorf("advancing usage based charge[%s] after collection_completed: %w", stateMachine.GetCharge().ID, err)
		}

		currentRun, err := stateMachine.GetCharge().GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}

		if err := populateUsageBasedStandardLineFromRun(stdLine, currentRun); err != nil {
			return nil, fmt.Errorf("populating standard line from run for charge[%s]: %w", stateMachine.GetCharge().ID, err)
		}
	}

	return input.Lines, nil
}

func (e *LineEngine) OnInvoiceIssued(ctx context.Context, input billing.OnInvoiceIssuedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.fireLineTrigger(ctx, fireLineTriggerInput{
		Lines:   input.Lines,
		Trigger: meta.TriggerInvoiceIssued,
		InputFn: func(stdLine *billing.StandardLine) any {
			return billing.StandardLineWithInvoiceHeader{
				Line:    stdLine,
				Invoice: input.Invoice,
			}
		},
		AdvanceUntilStateStable: true,
	})
}

func (e *LineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.recordRunPayments(ctx, recordRunPaymentsInput{
		Lines:    input.Lines,
		Invoice:  input.Invoice,
		RecordFn: e.recordPaymentAuthorized,
	})
}

func (e *LineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.recordRunPayments(ctx, recordRunPaymentsInput{
		Lines:    input.Lines,
		Invoice:  input.Invoice,
		RecordFn: e.recordPaymentSettled,
	})
}

func (e *LineEngine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("lines are required")
	}

	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil {
			return nil, fmt.Errorf("usage based standard line[%s]: charge id is required", stdLine.ID)
		}

		generatedDetailedLines, err := e.service.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}

func populateUsageBasedStandardLineFromRun(stdLine *billing.StandardLine, run usagebased.RealizationRun) error {
	if stdLine.UsageBased == nil {
		stdLine.UsageBased = &billing.UsageBasedLine{}
	}

	stdLine.UsageBased.Quantity = lo.ToPtr(run.MeterValue)
	stdLine.UsageBased.MeteredQuantity = lo.ToPtr(run.MeterValue)
	stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)

	creditsApplied, err := run.CreditsAllocated.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	return nil
}

type fireLineTriggerInput struct {
	Lines                   billing.StandardLines
	Trigger                 meta.Trigger
	InputFn                 func(*billing.StandardLine) any
	AdvanceUntilStateStable bool
}

func (i fireLineTriggerInput) Validate() error {
	if len(i.Lines) == 0 {
		return fmt.Errorf("lines are required")
	}

	if i.Trigger == "" {
		return fmt.Errorf("trigger is required")
	}

	if i.InputFn == nil {
		return fmt.Errorf("inputFn is required")
	}

	return nil
}

func (e *LineEngine) fireLineTrigger(ctx context.Context, input fireLineTriggerInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating fire line trigger input: %w", err)
	}

	for _, stdLine := range input.Lines {
		stateMachine, err := e.newStateMachineForStandardLine(ctx, stdLine)
		if err != nil {
			return err
		}

		canFire, err := stateMachine.CanFire(ctx, input.Trigger)
		if err != nil {
			return fmt.Errorf("checking %s for charge[%s]: %w", input.Trigger, stateMachine.GetCharge().ID, err)
		}

		if !canFire {
			return fmt.Errorf(
				"charge[%s] in status %s cannot handle %s for standard line[%s]",
				stateMachine.GetCharge().ID,
				stateMachine.GetCharge().Status,
				input.Trigger,
				stdLine.ID,
			)
		}

		if err := stateMachine.FireAndActivate(ctx, input.Trigger, input.InputFn(stdLine)); err != nil {
			return fmt.Errorf("triggering %s for charge[%s]: %w", input.Trigger, stateMachine.GetCharge().ID, err)
		}

		if input.AdvanceUntilStateStable {
			if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
				return fmt.Errorf("advancing usage based charge[%s] after %s: %w", stateMachine.GetCharge().ID, input.Trigger, err)
			}
		}
	}

	return nil
}

func (e *LineEngine) newStateMachineForStandardLine(ctx context.Context, stdLine *billing.StandardLine) (StateMachine, error) {
	if stdLine.ChargeID == nil {
		return nil, fmt.Errorf("usage based standard line[%s]: charge id is required", stdLine.ID)
	}

	charge, err := e.service.GetByID(ctx, usagebased.GetByIDInput{
		ChargeID: meta.ChargeID{
			Namespace: stdLine.Namespace,
			ID:        *stdLine.ChargeID,
		},
		Expands: meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return nil, fmt.Errorf("getting usage based charge for line[%s]: %w", stdLine.ID, err)
	}

	stateMachineConfig, err := e.service.getStateMachineConfigForPatch(ctx, charge)
	if err != nil {
		return nil, fmt.Errorf("getting state machine config for line[%s]: %w", stdLine.ID, err)
	}

	stateMachine, err := e.service.newStateMachine(stateMachineConfig)
	if err != nil {
		return nil, fmt.Errorf("creating state machine for line[%s]: %w", stdLine.ID, err)
	}

	return stateMachine, nil
}

func isUsageBasedSplitPeriodEmpty(line billing.GatheringLine) (bool, error) {
	price := line.GetPrice()
	if price == nil {
		return false, fmt.Errorf("price is nil")
	}

	if price.Type() == productcatalog.FlatPriceType {
		return false, nil
	}

	return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil
}
