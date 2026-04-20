package invoicecalc

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func RecalculateDetailedLinesAndTotals(invoice *billing.StandardInvoice, deps CalculatorDependencies) error {
	if invoice.Lines.IsAbsent() {
		return errors.New("cannot recaulculate invoice without expanded lines")
	}

	if deps.LineEngines == nil {
		return errors.New("line engines are nil")
	}

	var outErr error
	groupedLines := make(map[billing.LineEngineType]billing.StandardLines)

	for _, line := range invoice.Lines.OrEmpty() {
		if line.IsDeleted() {
			continue
		}

		groupedLines[line.Engine] = append(groupedLines[line.Engine], line)
	}

	for lineEngineType, stdLines := range groupedLines {
		lineEngine, err := deps.LineEngines.Get(lineEngineType)
		if err != nil {
			return fmt.Errorf("getting line engine[%s]: %w", lineEngineType, err)
		}

		input := billing.CalculateLinesInput{
			Invoice: *invoice,
			Lines:   stdLines,
		}
		if err := input.Validate(); err != nil {
			return fmt.Errorf("validating calculate lines input for engine[%s]: %w", lineEngineType, err)
		}

		updatedLines, err := lineEngine.CalculateLines(input)
		if err != nil {
			return fmt.Errorf("calculating lines for engine[%s]: %w", lineEngineType, err)
		}

		if err := updatedLines.Validate(); err != nil {
			return fmt.Errorf("validating calculate lines output for engine[%s]: %w", lineEngineType, err)
		}

		if err := billing.ValidateStandardLineIDsMatchExactly(stdLines, updatedLines); err != nil {
			return fmt.Errorf("validating calculate lines ids for engine[%s]: %w", lineEngineType, err)
		}

		if err := invoice.Lines.ReplaceLinesByID(updatedLines...); err != nil {
			return fmt.Errorf("replacing recalculated lines for engine[%s]: %w", lineEngineType, err)
		}
	}

	invoice.Totals = totals.Sum(
		lo.Map(invoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) totals.Totals {
			// Deleted lines are not contributing to the totals
			if line.DeletedAt != nil {
				return totals.Totals{}
			}

			return line.Totals
		})...,
	)

	return outErr
}

func newDetailedLines(line *billing.StandardLine, inputs ...rating.DetailedLine) (billing.DetailedLines, error) {
	return slicesx.MapWithErr(inputs, func(in rating.DetailedLine) (billing.DetailedLine, error) {
		if err := in.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		period := line.Period
		if in.Period != nil {
			period = timeutil.ClosedPeriod{
				From: in.Period.From,
				To:   in.Period.To,
			}
		}

		if in.Category == "" {
			in.Category = billing.FlatFeeCategoryRegular
		}

		line := billing.DetailedLine{
			DetailedLineBase: billing.DetailedLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: line.Namespace,
					Name:      in.Name,
				}),

				ServicePeriod:          period,
				InvoiceID:              line.InvoiceID,
				Currency:               line.Currency,
				ChildUniqueReferenceID: &in.ChildUniqueReferenceID,
				TaxConfig:              line.TaxConfig,

				PaymentTerm:    lo.CoalesceOrEmpty(in.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				PerUnitAmount:  in.PerUnitAmount,
				Quantity:       in.Quantity,
				Category:       in.Category,
				CreditsApplied: in.CreditsApplied,
				Totals:         in.Totals,
			},
			AmountDiscounts: in.AmountDiscounts,
		}

		if err := line.Validate(); err != nil {
			return billing.DetailedLine{}, err
		}

		return line, nil
	})
}

func MergeGeneratedDetailedLines(parentLine *billing.StandardLine, in rating.GenerateDetailedLinesResult) error {
	detailedLines, err := newDetailedLines(parentLine, in.DetailedLines...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	// The lines are generated in order, so we can just persist the index
	for idx := range detailedLines {
		detailedLines[idx].Index = lo.ToPtr(idx)
	}

	parentLine.DetailedLines = parentLine.DetailedLinesWithIDReuse(detailedLines)

	// Let's persist the other generation results
	parentLine.Totals = in.Totals
	if in.FinalUsage != nil {
		parentLine.UsageBased.Quantity = lo.ToPtr(in.FinalUsage.Quantity)
		parentLine.UsageBased.PreLinePeriodQuantity = lo.ToPtr(in.FinalUsage.PreLinePeriodQuantity)
	}

	parentLine.Discounts = in.FinalStandardLineDiscounts

	return nil
}
