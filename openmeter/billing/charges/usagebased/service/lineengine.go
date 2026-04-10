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

	if input.ProgressiveBilling {
		// TODO[later]: support progressive billing for usage-based charge lines once the
		// collection-complete lifecycle is wired through the usage-based charge state machine.
		return false, nil
	}

	return !input.AsOf.Before(input.ResolvedBillablePeriod.To), nil
}

func (e *LineEngine) SplitGatheringLine(_ context.Context, _ billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("usage based charge line is not progressively billed")
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

	stdLines, err = slicesx.MapWithErr(stdLines, func(stdLine *billing.StandardLine) (*billing.StandardLine, error) {
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

		// Becoming active after the service period starts is not an invoice lifecycle event, so we
		// still rely on the generic TriggerNext/AdvanceUntilStateStable flow before invoice-created
		// lifecycle transitions take over.
		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s]: %w", charge.ID, err)
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerInvoiceCreated); err != nil {
			return nil, fmt.Errorf("triggering invoice_created for charge[%s]: %w", charge.ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s] after invoice_created: %w", charge.ID, err)
		}

		currentRun, err := stateMachine.Charge.GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", charge.ID, err)
		}

		populateUsageBasedStandardLineFromRun(stdLine, currentRun)

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
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice is required")
	}

	for _, stdLine := range input.Lines {
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

		canFire, err := stateMachine.StateMachine.CanFireCtx(ctx, meta.TriggerCollectionCompleted)
		if err != nil {
			return nil, fmt.Errorf("checking collection_completed for charge[%s]: %w", charge.ID, err)
		}

		if !canFire {
			continue
		}

		if err := stateMachine.FireAndActivate(ctx, meta.TriggerCollectionCompleted); err != nil {
			return nil, fmt.Errorf("triggering collection_completed for charge[%s]: %w", charge.ID, err)
		}

		if _, err := stateMachine.AdvanceUntilStateStable(ctx); err != nil {
			return nil, fmt.Errorf("advancing usage based charge[%s] after collection_completed: %w", charge.ID, err)
		}

		currentRun, err := stateMachine.Charge.GetCurrentRealizationRun()
		if err != nil {
			return nil, fmt.Errorf("getting current realization run for charge[%s]: %w", charge.ID, err)
		}

		populateUsageBasedStandardLineFromRun(stdLine, currentRun)
	}

	return input.Lines, nil
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

func populateUsageBasedStandardLineFromRun(stdLine *billing.StandardLine, run usagebased.RealizationRun) {
	if stdLine.UsageBased == nil {
		stdLine.UsageBased = &billing.UsageBasedLine{}
	}

	stdLine.UsageBased.Quantity = lo.ToPtr(run.MeterValue)
	stdLine.UsageBased.MeteredQuantity = lo.ToPtr(run.MeterValue)
	stdLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
	stdLine.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.Zero)
}
