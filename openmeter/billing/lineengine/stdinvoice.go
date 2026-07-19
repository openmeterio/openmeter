package lineengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
)

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return e.buildStandardInvoiceLinesWithQuantitySnapshot(ctx, input)
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

	linesWithSplitLineHierarchy, err := e.ResolveSplitLineGroupHeaders(ctx, input.Invoice.Namespace, stdLines)
	if err != nil {
		return nil, fmt.Errorf("resolving split line group headers: %w", err)
	}

	if err := e.SnapshotLineQuantities(ctx, input.Invoice, linesWithSplitLineHierarchy); err != nil {
		return nil, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	calculatedLines, err := e.calculateLines(calculateLinesInput{
		Invoice: input.Invoice,
		Lines:   linesWithSplitLineHierarchy,
	})
	if err != nil {
		return nil, err
	}

	return calculatedLines.AsStandardLines(), nil
}

type calculateLinesInput struct {
	Invoice billing.StandardInvoice
	Lines   StandardLinesWithSplitLineHierarchy
}

func (i calculateLinesInput) Validate() error {
	var errs []error

	if i.Invoice.ID == "" {
		errs = append(errs, fmt.Errorf("invoice id is required"))
	}

	if len(i.Lines) == 0 {
		errs = append(errs, fmt.Errorf("lines are required"))
	}

	if err := i.Lines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("lines: %w", err))
	}

	return errors.Join(errs...)
}

// TODO: Do not implement the calculator here, so that we don't need to pass the splitline groups to the rating service
func (e *Engine) calculateLines(input calculateLinesInput) (StandardLinesWithSplitLineHierarchy, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		generatedDetailedLines, err := e.ratingService.GenerateProgressiveBilledDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine.StandardLine, generatedDetailedLines); err != nil {
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
