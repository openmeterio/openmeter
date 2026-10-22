package lineengine

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type RecalculateStandardLineInput struct {
	Invoice *billing.StandardInvoice
	Line    *billing.StandardLine
}

func (i RecalculateStandardLineInput) Validate() error {
	var errs []error
	if i.Invoice == nil {
		errs = append(errs, errors.New("invoice is required"))
	}

	if i.Line == nil {
		errs = append(errs, errors.New("line is required"))
	} else if i.Line.Engine != billing.LineEngineTypeInvoice {
		errs = append(errs, errors.New("line must be owned by the legacy invoice engine"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// RecalculateStandardLine completes a mutable legacy line on a clone. Snapshot
// failures return no replacement; rating issues accompany the calculated result.
// Immutable-invoice comparisons must continue using SnapshotLineQuantity.
func (e *Engine) RecalculateStandardLine(ctx context.Context, input RecalculateStandardLineInput) (*billing.StandardLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	line, err := input.Line.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning standard line: %w", err)
	}

	resolvedLines, err := e.resolveSplitLineHierarchies(ctx, input.Invoice.Namespace, billing.StandardLines{line})
	if err != nil {
		return nil, fmt.Errorf("resolving standard line hierarchy: %w", err)
	}

	if err := e.snapshotLineQuantities(ctx, *input.Invoice, resolvedLines); err != nil {
		return nil, fmt.Errorf("snapshotting standard line: %w", err)
	}

	lines, err := e.rateStandardLines(resolvedLines)
	if len(lines) == 0 {
		return nil, err
	}

	return lines[0], err
}

type RateStandardLinesWithSplitLineHierarchiesInput struct {
	Lines       billing.StandardLines
	Hierarchies map[billing.LineID]*billing.SplitLineHierarchy
}

func (i RateStandardLinesWithSplitLineHierarchiesInput) Validate() error {
	var errs []error
	if len(i.Lines) == 0 {
		errs = append(errs, errors.New("lines are required"))
	}

	for idx, line := range i.Lines {
		if line == nil {
			errs = append(errs, fmt.Errorf("lines[%d]: line is required", idx))
			continue
		}

		if line.UsageBased == nil {
			errs = append(errs, fmt.Errorf("lines[%d]: usage based line is required", idx))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// RateStandardLinesWithSplitLineHierarchies rates transient snapshots with supplied
// read-only history keyed by namespaced line ID. It performs no group loading,
// snapshotting or persistence and preserves calculated lines alongside rating issues.
func (e *Engine) RateStandardLinesWithSplitLineHierarchies(input RateStandardLinesWithSplitLineHierarchiesInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return e.rateStandardLines(lo.Map(input.Lines, func(line *billing.StandardLine, _ int) lineWithHierarchy {
		var hierarchy *billing.SplitLineHierarchy
		if line.SplitLineGroupID != nil {
			hierarchy = input.Hierarchies[line.GetLineID()]
		}

		return lineWithHierarchy{StandardLine: line, hierarchy: hierarchy}
	}))
}

func (e *Engine) rateStandardLines(lines linesWithHierarchy) (billing.StandardLines, error) {
	if len(lines) == 0 {
		return nil, errors.New("lines are required")
	}

	for idx, line := range lines {
		if line.StandardLine == nil {
			return nil, fmt.Errorf("lines[%d]: line is required", idx)
		}

		if line.UsageBased == nil {
			return nil, fmt.Errorf("lines[%d]: usage based line is required", idx)
		}
	}

	ratedLines, err := lines.Clone()
	if err != nil {
		return nil, fmt.Errorf("cloning standard lines: %w", err)
	}

	validationRecorder := billing.ValidationIssueRecorder{}

	for _, line := range ratedLines {
		line.RateCardDiscounts = line.RateCardDiscounts.UpsertCorrelationIDs()

		generatedDetailedLines, err := e.ratingService.GenerateLegacyProgressiveBilledLines(line)
		if err := validationRecorder.Record(err, billing.WithAttributes(models.Annotations{
			billing.AttributeKeyLineID: line.ID,
		})); err != nil {
			return nil, fmt.Errorf("calculating detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := mergeGeneratedDetailedLines(line.StandardLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging generated detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := line.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", line.ID, err)
		}
	}

	return ratedLines.MapStandardLines(), validationRecorder.ErrorsOrNil()
}

func newDetailedLines(stdLine *billing.StandardLine, inputs ...rating.DetailedLine) (billing.DetailedLines, error) {
	return slicesx.MapWithErr(inputs, func(in rating.DetailedLine) (billing.DetailedLine, error) {
		if err := in.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		period := stdLine.Period
		if in.Period != nil {
			period = timeutil.ClosedPeriod{
				From: in.Period.From,
				To:   in.Period.To,
			}
		}

		if in.Category == "" {
			in.Category = stddetailedline.CategoryRegular
		}

		detailedLine := billing.DetailedLine{
			InvoiceID:              stdLine.InvoiceID,
			Namespace:              stdLine.Namespace,
			Name:                   in.Name,
			ServicePeriod:          period,
			ChildUniqueReferenceID: in.ChildUniqueReferenceID,
			PaymentTerm:            lo.CoalesceOrEmpty(in.PaymentTerm, productcatalog.InArrearsPaymentTerm),
			PerUnitAmount:          in.PerUnitAmount,
			Quantity:               in.Quantity,
			Category:               in.Category,
			Totals:                 in.Totals,
			AmountDiscounts:        in.AmountDiscounts,
		}

		if err := detailedLine.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		return detailedLine, nil
	})
}

func mergeGeneratedDetailedLines(parentLine *billing.StandardLine, in rating.GenerateDetailedLinesResult) error {
	detailedLines, err := newDetailedLines(parentLine, in.DetailedLines...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	// The lines are generated in order, so we can just persist the index
	for idx := range detailedLines {
		detailedLines[idx].Index = lo.ToPtr(idx)
	}

	parentLine.DetailedLines = parentLine.DetailedLinesWithIDReuse(detailedLines)

	parentLine.Totals = in.Totals
	if in.FinalUsage != nil {
		parentLine.UsageBased.Quantity = lo.ToPtr(in.FinalUsage.Quantity)
		parentLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(in.FinalUsage.PreLinePeriodQuantity)
	}

	parentLine.Discounts = in.FinalStandardLineDiscounts

	return nil
}
