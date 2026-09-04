package lineengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := e.materializeStandardInvoiceLines(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := e.SnapshotLineQuantities(ctx, input.Invoice, stdLines); err != nil {
		return stdLines, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	stdLines, err = e.CalculateLines(billing.CalculateLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	})
	if err != nil {
		return nil, fmt.Errorf("calculating standard invoice lines: %w", err)
	}

	return stdLines, nil
}

func (e *Engine) BuildStandardLinesForGatheringPreview(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := e.materializeStandardInvoiceLines(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := e.SnapshotLineQuantities(ctx, input.Invoice, stdLines); err != nil {
		return nil, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	return stdLines, nil
}

func (e *Engine) materializeStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
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

func (e *Engine) AreLinesBillableAsOf(ctx context.Context, input billing.AreLinesBillableAsOfInput) ([]billing.IsLineBillableAsOfResult, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	featureMeters, err := e.featureMeterResolver.Resolve(ctx, input.Invoice.Namespace, input.Lines...)
	if err != nil {
		return nil, fmt.Errorf("resolving feature meters: %w", err)
	}

	return slicesx.MapWithErrPreservingResults(input.Lines, func(line billing.GatheringLine, _ int) (billing.IsLineBillableAsOfResult, error) {
		var errs []error
		ratingInput := rating.ResolveBillablePeriodInput{
			Line:               line,
			ProgressiveBilling: input.ProgressiveBilling,
			AsOf:               input.AsOf,
		}

		if line.GetFeatureMeterRef() != nil {
			featureMeter, err := featureMeters.Get(line)
			if err != nil {
				errs = append(errs, err)
			} else {
				ratingInput.Feature = &featureMeter.Feature
				ratingInput.Meter = featureMeter.Meter
			}
		}

		result, err := e.ratingService.ResolveBillablePeriod(ratingInput)
		if err != nil {
			errs = append(errs, fmt.Errorf("resolving billable period for line[%s]: %w", line.ID, err))
		}

		return result, errors.Join(errs...)
	})
}
