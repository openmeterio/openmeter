package lineservice

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ Line = (*feeLine)(nil)

type feeLine struct {
	lineBase
}

func (l feeLine) PrepareForCreate(context.Context) (Line, error) {
	return &l, nil
}

func (l feeLine) CanBeInvoicedAsOf(_ context.Context, in CanBeInvoicedAsOfInput) (*billing.Period, error) {
	// TODO[OM-1085]: Prorate can be implemented here for progressive billing/pro-rating of the fee

	if !in.AsOf.Before(l.line.InvoiceAt) {
		return &l.line.Period, nil
	}

	return nil, nil
}

func (l feeLine) SnapshotQuantity(context.Context, *billing.Invoice) error {
	return nil
}

func (l feeLine) CalculateDetailedLines() error {
	return nil
}

func (l *feeLine) UpdateTotals() error {
	// Calculate the line totals
	calc, err := l.line.Currency.Calculator()
	if err != nil {
		return err
	}

	// Calculate the line totals
	totals := billing.Totals{
		DiscountsTotal: calc.RoundToPrecision(
			alpacadecimal.Sum(alpacadecimal.Zero,
				lo.Map(l.line.Discounts.OrEmpty(), func(d billing.LineDiscount, _ int) alpacadecimal.Decimal {
					return d.Amount
				})...,
			),
		),

		// TODO[OM-979]: implement taxes
		TaxesInclusiveTotal: alpacadecimal.Zero,
		TaxesExclusiveTotal: alpacadecimal.Zero,
		TaxesTotal:          alpacadecimal.Zero,
	}

	amount := calc.RoundToPrecision(l.line.FlatFee.PerUnitAmount.Mul(l.line.FlatFee.Quantity))

	switch l.line.FlatFee.Category {
	case billing.FlatFeeCategoryCommitment:
		totals.ChargesTotal = amount
	default:
		totals.Amount = amount
	}

	totals.Total = totals.CalculateTotal()

	l.line.LineBase.Totals = totals
	return nil
}
