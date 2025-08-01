package lineservice

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

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

func (l feeLine) SnapshotQuantity(context.Context, []string) error {
	return nil
}

func (l feeLine) CalculateDetailedLines() error {
	// Fee lines only have percentage discounts, but no commitments, so it's fine to not to reuse the whole
	// middleware line for now.
	pctDiscount, err := l.getPercentageDiscounts()
	if err != nil {
		return err
	}

	// The merge should happen in an idempotent way, or we end up with multiple discounts for the same line
	// due to recalculations.

	targetDiscountState := billing.LineDiscounts{}

	if pctDiscount != nil {
		targetDiscountState.Amount = append(targetDiscountState.Amount, *pctDiscount)
	}

	l.line.Discounts, err = targetDiscountState.ReuseIDsFrom(l.line.Discounts)
	if err != nil {
		return err
	}

	return nil
}

func (l feeLine) getPercentageDiscounts() (*billing.AmountLineDiscountManaged, error) {
	discountPercentageMutator := discountPercentageMutator{}

	discount, err := discountPercentageMutator.getDiscount(l.line.RateCardDiscounts)
	if err != nil {
		return nil, err
	}

	if discount == nil {
		return nil, nil
	}

	currencyCalc, err := l.line.Currency.Calculator()
	if err != nil {
		return nil, err
	}

	amount := TotalAmount(getTotalAmountInput{
		Currency:      currencyCalc,
		PerUnitAmount: l.line.FlatFee.PerUnitAmount,
		Quantity:      l.line.FlatFee.Quantity,
	})

	lineDiscount, err := discountPercentageMutator.getLineDiscount(amount, currencyCalc, *discount)
	if err != nil {
		return nil, err
	}

	return &lineDiscount, nil
}

func (l *feeLine) UpdateTotals() error {
	// Calculate the line totals
	calc, err := l.line.Currency.Calculator()
	if err != nil {
		return err
	}

	// Calculate the line totals
	totals := billing.Totals{
		DiscountsTotal: l.line.Discounts.Amount.SumAmount(calc),

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

func (l feeLine) IsPeriodEmptyConsideringTruncations() bool {
	// Fee lines are not subject to truncation, and for now they can be empty (one time fees)
	return false
}
