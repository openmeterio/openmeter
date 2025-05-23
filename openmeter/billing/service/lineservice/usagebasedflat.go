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

func (l ubpFlatFeeLine) PrepareForCreate(context.Context) (Line, error) {
	return &l, nil
}

func (l ubpFlatFeeLine) Validate(ctx context.Context, targetInvoice *billing.Invoice) error {
	var outErr []error

	if l.line.UsageBased.FeatureKey != "" {
		if _, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey); err != nil {
			outErr = append(outErr, err)
		}
	}

	if err := l.lineBase.Validate(ctx, targetInvoice); err != nil {
		outErr = append(outErr, err)
	}

	// Metered fields are not allowed

	if l.line.UsageBased.MeteredPreLinePeriodQuantity != nil {
		outErr = append(outErr, errors.New("metered pre line period quantity is not supported for usage based flat fee lines"))
	}

	if l.line.UsageBased.PreLinePeriodQuantity != nil {
		outErr = append(outErr, errors.New("pre line period quantity is not supported for usage based flat fee lines"))
	}

	if l.line.UsageBased.MeteredQuantity != nil {
		outErr = append(outErr, errors.New("metered quantity is not supported for usage based flat fee lines"))
	}

	// Quantity must be pre-set, as it's the only way to know how much to charge for the line
	if l.line.UsageBased.Quantity == nil {
		outErr = append(outErr, errors.New("quantity is required for usage based flat fee lines"))
	} else {
		if l.line.UsageBased.Quantity.IsNegative() {
			outErr = append(outErr, errors.New("quantity must be positive"))
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

func (l ubpFlatFeeLine) SnapshotQuantity(context.Context, *billing.Invoice) error {
	// Flat fee lines already have a quantity set, so we don't need to snapshot anything

	return nil
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
