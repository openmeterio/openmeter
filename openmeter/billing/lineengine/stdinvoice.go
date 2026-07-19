package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
)

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := e.buildStandardInvoiceLinesWithQuantitySnapshot(ctx, input)
	if err != nil {
		return nil, err
	}

	return e.CalculateLines(billing.CalculateLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	})
}

func (e *Engine) BuildStandardLinesForGatheringPreview(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return e.buildStandardInvoiceLinesWithQuantitySnapshot(ctx, input)
}

func (e *Engine) buildStandardInvoiceLinesWithQuantitySnapshot(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.GatheringLines) == 0 {
		return nil, fmt.Errorf("gathering lines are required")
	}

	stdLines, err := input.GatheringLines.ToStandardLines(input.Invoice.ID)
	if err != nil {
		return nil, err
	}

	if err := e.ResolveSplitLineGroupHeaders(ctx, input.Invoice.Namespace, stdLines); err != nil {
		return nil, fmt.Errorf("resolving split line group headers: %w", err)
	}

	if err := e.SnapshotLineQuantities(ctx, input.Invoice, stdLines); err != nil {
		return nil, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	return stdLines, nil
}

func (e *Engine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("lines are required")
	}

	for _, stdLine := range input.Lines {
		generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
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

func (e *Engine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return !lo.IsEmpty(input.ResolvedBillablePeriod), nil
}
