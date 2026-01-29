package lineservice

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ Line = (*ubpFlatFeeLine)(nil)

type ubpFlatFeeLine struct {
	lineBase
}

func (l ubpFlatFeeLine) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*billing.Period, error) {
	if !in.AsOf.Before(l.line.InvoiceAt) {
		return &l.line.Period, nil
	}

	return nil, nil
}

func (l ubpFlatFeeLine) calculateDetailedLines() (newDetailedLinesInput, error) {
	pricer := &priceMutator{
		Pricer: flatPricer{},
		PostCalculation: []PostCalculationMutator{
			&discountPercentageMutator{},
		},
	}

	return pricer.Calculate(PricerCalculateInput(l))
}

func (l ubpFlatFeeLine) CalculateDetailedLines() error {
	newDetailedLinesInput, err := l.calculateDetailedLines()
	if err != nil {
		return err
	}

	if err := mergeDetailedLines(l.line, newDetailedLinesInput); err != nil {
		return fmt.Errorf("merging detailed lines: %w", err)
	}

	return nil
}

func (l *ubpFlatFeeLine) UpdateTotals() error {
	return UpdateTotalsFromDetailedLines(l.line)
}

func (l ubpFlatFeeLine) IsPeriodEmptyConsideringTruncations() bool {
	// Fee lines are not subject to truncation, and for now they can be empty (one time fees)
	return false
}
