package lineservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ Line = (*ubpFlatFeeLine)(nil)

type ubpFlatFeeLine struct {
	lineBase
}

func (l ubpFlatFeeLine) Validate(ctx context.Context, targetInvoice *billing.StandardInvoice) error {
	var outErr []error

	if l.line.UsageBased.FeatureKey != "" {
		_, err := l.featureMeters.Get(l.line.UsageBased.FeatureKey, false)
		if err != nil {
			outErr = append(outErr, fmt.Errorf("fetching feature[%s]: %w", l.line.UsageBased.FeatureKey, err))
		}
	}

	// Usage discounts are not allowed
	// TODO[later]: Once we have cleaned up the line types, let's move as much as possible to the line's validation
	if l.line.RateCardDiscounts.Usage != nil {
		outErr = append(outErr, errors.New("usage discounts are not supported for usage based flat fee lines"))
	}

	// Percentage discounts are allowed
	if l.line.RateCardDiscounts.Percentage != nil {
		if err := l.line.RateCardDiscounts.Percentage.Validate(); err != nil {
			outErr = append(outErr, err)
		}
	}

	return errors.Join(outErr...)
}

func (l ubpFlatFeeLine) CanBeInvoicedAsOf(_ context.Context, in CanBeInvoicedAsOfInput) (*billing.Period, error) {
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
	return l.service.UpdateTotalsFromDetailedLines(l.line)
}

func (l ubpFlatFeeLine) IsPeriodEmptyConsideringTruncations() bool {
	// Fee lines are not subject to truncation, and for now they can be empty (one time fees)
	return false
}
