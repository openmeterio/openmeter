package lineengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	resolvedLines, err := e.materializeStandardInvoiceLines(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := e.snapshotLineQuantities(ctx, input.Invoice, resolvedLines); err != nil {
		return resolvedLines.MapStandardLines(), fmt.Errorf("snapshotting line quantities: %w", err)
	}

	recorder := billing.ValidationIssueRecorder{}

	stdLines, err := e.rateStandardLines(resolvedLines)
	if err := recorder.Record(err); err != nil {
		return nil, fmt.Errorf("calculating standard invoice lines: %w", err)
	}

	return stdLines, recorder.ErrorsOrNil()
}

func (e *Engine) BuildStandardLinesForGatheringPreview(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	resolvedLines, err := e.materializeStandardInvoiceLines(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := e.snapshotLineQuantities(ctx, input.Invoice, resolvedLines); err != nil {
		return nil, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	return e.rateStandardLines(resolvedLines)
}

func (e *Engine) materializeStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (linesWithHierarchy, error) {
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

	resolvedLines, err := e.resolveSplitLineHierarchies(ctx, input.Invoice.Namespace, stdLines)
	if err != nil {
		return nil, fmt.Errorf("resolving split line hierarchies: %w", err)
	}

	// TODO[later]: This is a temporary workaround until we remove the SplitLineHierarchy field from the StandardLine type.
	for _, line := range resolvedLines {
		if line.SplitLineGroupID != nil {
			// Retain the existing returned expansion independently of calculation input.
			line.SplitLineHierarchy = line.hierarchy
		}
	}

	return resolvedLines, nil
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

func (e *Engine) GateInvoiceAssignment(context.Context, billing.GateInvoiceAssignmentInput) (billing.GateInvoiceAssignmentResult, error) {
	return nil, nil
}
