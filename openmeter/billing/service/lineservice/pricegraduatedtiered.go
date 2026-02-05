package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type graduatedTieredPricer struct {
	ProgressiveBillingMeteredPricer
}

var _ Pricer = (*graduatedTieredPricer)(nil)

func (p graduatedTieredPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to graduated tiered price: %w", err)
	}

	if price.Mode != productcatalog.GraduatedTieredPrice {
		return nil, fmt.Errorf("price is not a graduated tiered price")
	}

	out := make(newDetailedLinesInput, 0, len(price.Tiers))

	err = p.tieredPriceCalculator(tieredPriceCalculatorInput{
		TieredPrice: price,
		FromQty:     usage.PreLinePeriodQuantity,
		ToQty:       usage.LinePeriodQuantity.Add(usage.PreLinePeriodQuantity),
		Currency:    l.currency,
		TierCallbackFn: func(in tierCallbackInput) error {
			tierIndex := in.TierIndex + 1

			if in.Tier.UnitPrice != nil {
				newLine := newDetailedLineInput{
					Name:                   fmt.Sprintf("%s: usage price for tier %d", l.line.Name, tierIndex),
					Quantity:               in.Quantity,
					PerUnitAmount:          in.Tier.UnitPrice.Amount,
					ChildUniqueReferenceID: fmt.Sprintf(GraduatedTieredPriceUsageChildUniqueReferenceID, tierIndex),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				}

				out = append(out, newLine)
			}

			// If have already billed this flat price for the previous split line, so we can skip it
			shouldFirstFlatLineBeBilled := in.TierIndex > 0 || l.IsFirstInPeriod()

			// Flat price is always billed for the whole tier when we are crossing the tier boundary
			if in.Tier.FlatPrice != nil && in.AtTierBoundary && shouldFirstFlatLineBeBilled {
				newLine := newDetailedLineInput{
					Name:                   fmt.Sprintf("%s: flat price for tier %d", l.line.Name, tierIndex),
					Quantity:               alpacadecimal.NewFromFloat(1),
					PerUnitAmount:          in.Tier.FlatPrice.Amount,
					ChildUniqueReferenceID: fmt.Sprintf(GraduatedTieredFlatPriceChildUniqueReferenceID, tierIndex),
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				}

				out = append(out, newLine)
			}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calculating tiered price: %w", err)
	}

	return out, nil
}

type tierRange struct {
	Tier      productcatalog.PriceTier
	TierIndex int

	FromQty        alpacadecimal.Decimal // exclusive
	ToQty          alpacadecimal.Decimal // inclusive
	AtTierBoundary bool
}

type tierCallbackInput struct {
	Tier                productcatalog.PriceTier
	TierIndex           int
	Quantity            alpacadecimal.Decimal
	AtTierBoundary      bool
	PreviousTotalAmount alpacadecimal.Decimal
}

type tieredPriceCalculatorInput struct {
	TieredPrice productcatalog.TieredPrice
	// FromQty is the quantity that was already billed for the previous tiers (exclusive)
	FromQty alpacadecimal.Decimal
	// ToQty is the quantity that we are going to bill for this tiered price (inclusive)
	ToQty alpacadecimal.Decimal

	Currency currencyx.Calculator

	TierCallbackFn     func(tierCallbackInput) error
	FinalizerFn        func(total alpacadecimal.Decimal) error
	IntrospectRangesFn func(ranges []tierRange)
}

func (i tieredPriceCalculatorInput) Validate() error {
	if err := i.TieredPrice.Validate(); err != nil {
		return err
	}

	if i.TieredPrice.Mode != productcatalog.GraduatedTieredPrice {
		return fmt.Errorf("only graduated tiered prices are supported")
	}

	if i.FromQty.IsNegative() {
		return fmt.Errorf("from quantity must be zero or positive")
	}

	if i.ToQty.IsNegative() {
		return fmt.Errorf("to quantity must be zero or positive")
	}

	if i.ToQty.LessThan(i.FromQty) {
		return fmt.Errorf("to quantity must be greater or equal to from quantity")
	}

	if i.Currency.Currency == "" {
		return fmt.Errorf("currency calculator is required")
	}

	return nil
}

func (p graduatedTieredPricer) splitTierRangeAtBoundary(from, to alpacadecimal.Decimal, qtyRange tierRange) []tierRange {
	res := make([]tierRange, 0, 3)

	// Pending line is always the last line, as we might need to split it
	pendingLine := qtyRange

	// If from == in.FromQty we don't need to split the range, as the range is already at some boundary
	if pendingLine.FromQty.LessThan(from) && pendingLine.ToQty.GreaterThan(from) {
		// We need to split the range at the from boundary
		res = append(res, tierRange{
			Tier:      pendingLine.Tier,
			TierIndex: pendingLine.TierIndex,

			FromQty: pendingLine.FromQty,
			ToQty:   from,

			AtTierBoundary: pendingLine.AtTierBoundary,
		})

		pendingLine = tierRange{
			Tier:      pendingLine.Tier,
			TierIndex: pendingLine.TierIndex,

			FromQty: from,
			ToQty:   pendingLine.ToQty,
		}
	}

	// If to == in.ToQty we don't need to split the range, as the range is already at some boundary
	if pendingLine.FromQty.LessThan(to) && pendingLine.ToQty.GreaterThan(to) {
		res = append(res, tierRange{
			Tier:      pendingLine.Tier,
			TierIndex: pendingLine.TierIndex,

			FromQty: pendingLine.FromQty,
			ToQty:   to,

			AtTierBoundary: pendingLine.AtTierBoundary,
		})
		pendingLine = tierRange{
			Tier:      pendingLine.Tier,
			TierIndex: pendingLine.TierIndex,

			FromQty: to,
			ToQty:   pendingLine.ToQty,
		}
	}

	return append(res, pendingLine)
}

// getTotalAmountForGraduatedTieredPrice calculates the total amount for a graduated tiered price for a given quantity
// without considering any discounts
func (p graduatedTieredPricer) tieredPriceCalculator(in tieredPriceCalculatorInput) error {
	// Note: this is not the most efficient algorithm, but it is at least pseudo-readable
	if err := in.Validate(); err != nil {
		return err
	}

	// Let's break up the tiers and the input data into a sequence of periods, for easier processing
	// Invariant of the qtyRanges:
	// - Non overlapping ranges
	// - The ranges are sorted by the from quantity
	// - There is always one range for which range.From == in.FromQty
	// - There is always one range for which range.ToQty == in.ToQty
	qtyRanges := make([]tierRange, 0, len(in.TieredPrice.Tiers)+2)

	previousTierQty := alpacadecimal.Zero
	for idx, tier := range in.TieredPrice.WithSortedTiers().Tiers {
		if previousTierQty.GreaterThanOrEqual(in.ToQty) {
			// We already have enough data to bill for this tiered price
			break
		}

		// Given that the previous tier's max qty was less than then in.ToQty, toQty will fall into the
		// open ended tier, so we can safely use it as the upper bound
		tierUpperBound := in.ToQty
		if tier.UpToAmount != nil {
			tierUpperBound = *tier.UpToAmount
		}

		input := tierRange{
			Tier:           tier,
			TierIndex:      idx,
			AtTierBoundary: true,
			FromQty:        previousTierQty,
			ToQty:          tierUpperBound,
		}

		qtyRanges = append(qtyRanges, p.splitTierRangeAtBoundary(in.FromQty, in.ToQty, input)...)

		previousTierQty = tierUpperBound
	}

	if in.ToQty.Equal(alpacadecimal.Zero) {
		// We need to add the first range, in case there's a flat price component
		qtyRanges = append(qtyRanges, tierRange{
			Tier:           in.TieredPrice.Tiers[0],
			TierIndex:      0,
			AtTierBoundary: true,
			FromQty:        alpacadecimal.Zero,
			ToQty:          alpacadecimal.Zero,
		})
	}

	if in.IntrospectRangesFn != nil {
		in.IntrospectRangesFn(qtyRanges)
	}

	// Now that we have the ranges, let's iterate over the ranges and calculate the cummulative total amount
	// and call the callback for each in-scope range
	total := alpacadecimal.Zero
	shouldEmitCallbacks := false
	for _, qtyRange := range qtyRanges {
		if qtyRange.FromQty.Equal(in.FromQty) {
			shouldEmitCallbacks = true
		}

		if shouldEmitCallbacks && in.TierCallbackFn != nil {
			err := in.TierCallbackFn(tierCallbackInput{
				Tier:                qtyRange.Tier,
				TierIndex:           qtyRange.TierIndex,
				Quantity:            qtyRange.ToQty.Sub(qtyRange.FromQty),
				PreviousTotalAmount: total,
				AtTierBoundary:      qtyRange.AtTierBoundary,
			})
			if err != nil {
				return err
			}
		}

		// Let's update totals
		if qtyRange.Tier.FlatPrice != nil && qtyRange.AtTierBoundary {
			total = total.Add(in.Currency.RoundToPrecision(qtyRange.Tier.FlatPrice.Amount))
		}

		if qtyRange.Tier.UnitPrice != nil {
			total = total.Add(in.Currency.RoundToPrecision(qtyRange.ToQty.Sub(qtyRange.FromQty).Mul(qtyRange.Tier.UnitPrice.Amount)))
		}

		// We should only calculate totals up to in.ToQty (given tiers are open-ended we cannot have a full upper bound
		// either ways)
		if qtyRange.ToQty.GreaterThanOrEqual(in.ToQty) {
			break
		}
	}

	if in.FinalizerFn != nil {
		if err := in.FinalizerFn(total); err != nil {
			return err
		}
	}

	return nil
}
