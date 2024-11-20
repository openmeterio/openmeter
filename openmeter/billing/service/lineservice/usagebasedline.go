package lineservice

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ Line = usageBasedLine{}

const (
	FlatPriceChildUniqueReferenceID         = "flat-price"
	UnitPriceUsageChildUniqueReferenceID    = "unit-price-usage"
	UnitPriceMinSpendChildUniqueReferenceID = "unit-price-min-spend"
	UnitPriceMaxSpendChildUniqueReferenceID = "unit-price-max-spend"

	VolumeFlatPriceChildUniqueReferenceID = "volume-flat-price"
	VolumeUnitPriceChildUniqueReferenceID = "volume-tiered-price"
	VolumeMinSpendChildUniqueReferenceID  = "volume-min-spend"

	GraduatedTieredPriceUsageChildUniqueReferenceID = "graduated-tiered-%d-price-usage"
	GraduatedTieredFlatPriceChildUniqueReferenceID  = "graduated-tiered-%d-flat-price"
	GraduatedMinSpendChildUniqueReferenceID         = "graduated-tiered-min-spend"
)

type usageBasedLine struct {
	lineBase
}

func (l usageBasedLine) PrepareForCreate(context.Context) (Line, error) {
	l.line.Period = l.line.Period.Truncate(billingentity.DefaultMeterResolution)

	return l, nil
}

func (l usageBasedLine) Validate(ctx context.Context, targetInvoice *billingentity.Invoice) error {
	if _, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey); err != nil {
		return err
	}

	if err := l.lineBase.Validate(ctx, targetInvoice); err != nil {
		return err
	}

	if len(targetInvoice.Customer.UsageAttribution.SubjectKeys) == 0 {
		return billingentity.ValidationError{
			Err: billingentity.ErrInvoiceCreateUBPLineCustomerHasNoSubjects,
		}
	}

	if l.line.LineBase.Period.Truncate(billingentity.DefaultMeterResolution).IsEmpty() {
		return billingentity.ValidationError{
			Err: billingentity.ErrInvoiceCreateUBPLinePeriodIsEmpty,
		}
	}

	return nil
}

func (l usageBasedLine) CanBeInvoicedAsOf(ctx context.Context, asof time.Time) (*billingentity.Period, error) {
	if l.line.UsageBased.Price.Type() == plan.TieredPriceType {
		tiered, err := l.line.UsageBased.Price.AsTiered()
		if err != nil {
			return nil, err
		}

		if tiered.Mode == plan.VolumeTieredPrice {
			// Volume tiers are only billable if we have all the data acquired, as otherwise
			// we might overcharge the customer (if we are at tier bundaries)
			if !asof.Before(l.line.InvoiceAt) {
				return &l.line.Period, nil
			}
			return nil, nil
		}
	}

	meterAndFactory, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return nil, err
	}

	meter := meterAndFactory.meter

	asOfTruncated := asof.Truncate(billingentity.DefaultMeterResolution)

	switch meter.Aggregation {
	case models.MeterAggregationSum, models.MeterAggregationCount,
		models.MeterAggregationMax, models.MeterAggregationUniqueCount:

		periodStartTrucated := l.line.Period.Start.Truncate(billingentity.DefaultMeterResolution)

		if !periodStartTrucated.Before(asOfTruncated) {
			return nil, nil
		}

		candidatePeriod := billingentity.Period{
			Start: periodStartTrucated,
			End:   asOfTruncated,
		}

		if candidatePeriod.IsEmpty() {
			return nil, nil
		}

		return &candidatePeriod, nil
	default:
		// Other types need to be billed arrears truncated by window size
		if !asOfTruncated.Before(l.line.InvoiceAt) {
			return &l.line.Period, nil
		}
		return nil, nil
	}
}

func (l usageBasedLine) SnapshotQuantity(ctx context.Context, invoice *billingentity.Invoice) error {
	featureMeter, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return err
	}

	usage, err := l.service.getFeatureUsage(ctx,
		getFeatureUsageInput{
			Line:       l.line,
			ParentLine: l.line.ParentLine,
			Feature:    featureMeter.feature,
			Meter:      featureMeter.meter,
			Subjects:   invoice.Customer.UsageAttribution.SubjectKeys,
		},
	)
	if err != nil {
		return err
	}

	l.line.UsageBased.Quantity = lo.ToPtr(usage.LinePeriodQty)

	newDetailedLinesInput, err := l.calculateDetailedLines(usage)
	if err != nil {
		return err
	}

	detailedLines, err := l.newDetailedLines(newDetailedLinesInput...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	l.line.Children = l.line.ChildrenWithIDReuse(detailedLines)

	return nil
}

func (l usageBasedLine) calculateDetailedLines(usage *featureUsageResponse) (newDetailedLinesInput, error) {
	switch l.line.UsageBased.Price.Type() {
	case plan.FlatPriceType:
		flatPrice, err := l.line.UsageBased.Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat price: %w", err)
		}
		return l.calculateFlatPriceDetailedLines(usage, flatPrice)

	case plan.UnitPriceType:
		unitPrice, err := l.line.UsageBased.Price.AsUnit()
		if err != nil {
			return nil, fmt.Errorf("converting price to unit price: %w", err)
		}

		return l.calculateUnitPriceDetailedLines(usage, unitPrice)
	case plan.TieredPriceType:
		tieredPrice, err := l.line.UsageBased.Price.AsTiered()
		if err != nil {
			return nil, fmt.Errorf("converting price to tiered price: %w", err)
		}

		switch tieredPrice.Mode {
		case plan.VolumeTieredPrice:
			return l.calculateVolumeTieredPriceDetailedLines(usage, tieredPrice)

		case plan.GraduatedTieredPrice:
			return l.calculateGraduatedTieredPriceDetailedLines(usage, tieredPrice)
		default:
			return nil, fmt.Errorf("unsupported tiered price mode: %s", tieredPrice.Mode)
		}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", l.line.UsageBased.Price.Type())
	}
}

func (l usageBasedLine) calculateFlatPriceDetailedLines(_ *featureUsageResponse, flatPrice plan.FlatPrice) (newDetailedLinesInput, error) {
	// Flat price is the same as the non-metered version, we just allow attaching entitlements to it
	switch {
	case flatPrice.PaymentTerm == plan.InAdvancePaymentTerm && l.IsFirstInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            plan.InAdvancePaymentTerm,
			},
		}, nil
	case flatPrice.PaymentTerm != plan.InAdvancePaymentTerm && l.IsLastInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            plan.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

func (l usageBasedLine) calculateUnitPriceDetailedLines(usage *featureUsageResponse, unitPrice plan.UnitPrice) (newDetailedLinesInput, error) {
	out := make(newDetailedLinesInput, 0, 2)
	totalPreUsageAmount := usage.PreLinePeriodQty.Mul(unitPrice.Amount)

	if usage.LinePeriodQty.IsPositive() {
		usageLine := newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
			Quantity:               usage.LinePeriodQty,
			PerUnitAmount:          unitPrice.Amount,
			ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
			PaymentTerm:            plan.InArrearsPaymentTerm,
		}

		if unitPrice.MaximumAmount != nil {
			// We need to apply the discount for the usage that is over the maximum spend
			usageLine = usageLine.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: totalPreUsageAmount,
				MaxSpend:               *unitPrice.MaximumAmount,
			})
		}

		out = append(out, usageLine)
	}

	// Minimum spend is always billed arrears
	if l.IsLastInPeriod() && unitPrice.MinimumAmount != nil {
		totalUsageAmount := totalPreUsageAmount.Add(out.Sum())
		if totalUsageAmount.LessThan(*unitPrice.MinimumAmount) {
			period := l.line.Period
			if l.line.ParentLine != nil {
				period = l.line.ParentLine.Period
			}

			out = append(out, newDetailedLineInput{
				Name:          fmt.Sprintf("%s: minimum spend", l.line.Name),
				Quantity:      alpacadecimal.NewFromFloat(1),
				PerUnitAmount: unitPrice.MinimumAmount.Sub(totalUsageAmount),
				// Min spend is always billed for the whole period
				Period:                 &period,
				ChildUniqueReferenceID: UnitPriceMinSpendChildUniqueReferenceID,
				PaymentTerm:            plan.InArrearsPaymentTerm,
			})
		}
	}

	return out, nil
}

func (l usageBasedLine) calculateVolumeTieredPriceDetailedLines(usage *featureUsageResponse, price plan.TieredPrice) (newDetailedLinesInput, error) {
	if !usage.PreLinePeriodQty.IsZero() {
		return nil, billingentity.ErrInvoiceLineVolumeSplitNotSupported
	}

	if !l.IsLastInPeriod() {
		return nil, nil
	}

	out := make(newDetailedLinesInput, 0, 4)

	findTierRes, err := findTierForQuantity(price, usage.LinePeriodQty)
	if err != nil {
		return nil, err
	}

	tier := findTierRes.Tier
	tierIndex := findTierRes.Index

	if tier.FlatPrice != nil {
		line := newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: flat price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          tier.FlatPrice.Amount,
			ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
			PaymentTerm:            plan.InArrearsPaymentTerm,
		}

		if price.MaximumAmount != nil {
			line = line.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: out.Sum(),
				MaxSpend:               *price.MaximumAmount,
			})
		}
		out = append(out, line)
	}

	if tier.UnitPrice != nil && !usage.LinePeriodQty.IsZero() {
		line := newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: unit price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               usage.LinePeriodQty,
			PerUnitAmount:          tier.UnitPrice.Amount,
			ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
			PaymentTerm:            plan.InArrearsPaymentTerm,
		}

		if price.MaximumAmount != nil {
			line = line.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: out.Sum(),
				MaxSpend:               *price.MaximumAmount,
			})
		}

		out = append(out, line)
	}

	total := out.Sum()

	if price.MinimumAmount != nil && total.LessThan(*price.MinimumAmount) {
		out = append(out, newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: minimum spend", l.line.Name),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          price.MinimumAmount.Sub(total),
			ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
			PaymentTerm:            plan.InArrearsPaymentTerm,
		})
	}

	return out, nil
}

type findTierForQuantityResult struct {
	Tier  *plan.PriceTier
	Index int
}

func findTierForQuantity(price plan.TieredPrice, quantity alpacadecimal.Decimal) (findTierForQuantityResult, error) {
	for i, tier := range price.WithSortedTiers().Tiers {
		if tier.UpToAmount == nil || quantity.LessThanOrEqual(*tier.UpToAmount) {
			return findTierForQuantityResult{
				Tier:  &price.Tiers[i],
				Index: i,
			}, nil
		}
	}

	// Technically this should not happen, as the last tier should have an upper limit of infinity
	return findTierForQuantityResult{}, fmt.Errorf("could not find tier for quantity %s: %w", quantity, billingentity.ErrInvoiceLineMissingOpenEndedTier)
}

func (l usageBasedLine) calculateGraduatedTieredPriceDetailedLines(usage *featureUsageResponse, price plan.TieredPrice) (newDetailedLinesInput, error) {
	out := make(newDetailedLinesInput, 0, len(price.Tiers))

	err := tieredPriceCalculator(tieredPriceCalculatorInput{
		TieredPrice: price,
		FromQty:     usage.PreLinePeriodQty,
		ToQty:       usage.LinePeriodQty.Add(usage.PreLinePeriodQty),
		TierCallbackFn: func(in tierCallbackInput) error {
			billedAmount := in.PreviousTotalAmount

			tierIndex := in.TierIndex + 1

			if in.Tier.UnitPrice != nil {
				newLine := newDetailedLineInput{
					Name:                   fmt.Sprintf("%s: usage price for tier %d", l.line.Name, tierIndex),
					Quantity:               in.Quantity,
					PerUnitAmount:          in.Tier.UnitPrice.Amount,
					ChildUniqueReferenceID: fmt.Sprintf(GraduatedTieredPriceUsageChildUniqueReferenceID, tierIndex),
					PaymentTerm:            plan.InArrearsPaymentTerm,
				}

				if price.MaximumAmount != nil {
					newLine = newLine.AddDiscountForOverage(addDiscountInput{
						BilledAmountBeforeLine: billedAmount,
						MaxSpend:               *price.MaximumAmount,
					})
				}

				billedAmount = billedAmount.Add(in.Quantity.Mul(in.Tier.UnitPrice.Amount))

				out = append(out, newLine)
			}

			// Flat price is always billed for the whole tier when we are crossing the tier boundary
			if in.Tier.FlatPrice != nil && in.AtTierBoundary {
				newLine := newDetailedLineInput{
					Name:                   fmt.Sprintf("%s: flat price for tier %d", l.line.Name, tierIndex),
					Quantity:               alpacadecimal.NewFromFloat(1),
					PerUnitAmount:          in.Tier.FlatPrice.Amount,
					ChildUniqueReferenceID: fmt.Sprintf(GraduatedTieredFlatPriceChildUniqueReferenceID, tierIndex),
					PaymentTerm:            plan.InArrearsPaymentTerm,
				}

				if price.MaximumAmount != nil {
					newLine = newLine.AddDiscountForOverage(addDiscountInput{
						BilledAmountBeforeLine: billedAmount,
						MaxSpend:               *price.MaximumAmount,
					})
				}

				out = append(out, newLine)
			}
			return nil
		},
		FinalizerFn: func(periodTotal alpacadecimal.Decimal) error {
			if l.IsLastInPeriod() && price.MinimumAmount != nil && periodTotal.LessThan(*price.MinimumAmount) {
				out = append(out, newDetailedLineInput{
					Name:                   fmt.Sprintf("%s: minimum spend", l.line.Name),
					Quantity:               alpacadecimal.NewFromFloat(1),
					PerUnitAmount:          price.MinimumAmount.Sub(periodTotal),
					ChildUniqueReferenceID: GraduatedMinSpendChildUniqueReferenceID,
					PaymentTerm:            plan.InArrearsPaymentTerm,
				})
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
	Tier      plan.PriceTier
	TierIndex int

	FromQty        alpacadecimal.Decimal // exclusive
	ToQty          alpacadecimal.Decimal // inclusive
	AtTierBoundary bool
}

type tierCallbackInput struct {
	Tier                plan.PriceTier
	TierIndex           int
	Quantity            alpacadecimal.Decimal
	AtTierBoundary      bool
	PreviousTotalAmount alpacadecimal.Decimal
}

type tieredPriceCalculatorInput struct {
	TieredPrice plan.TieredPrice
	// FromQty is the quantity that was already billed for the previous tiers (exclusive)
	FromQty alpacadecimal.Decimal
	// ToQty is the quantity that we are going to bill for this tiered price (inclusive)
	ToQty alpacadecimal.Decimal

	TierCallbackFn     func(tierCallbackInput) error
	FinalizerFn        func(total alpacadecimal.Decimal) error
	IntrospectRangesFn func(ranges []tierRange)
}

func (i tieredPriceCalculatorInput) Validate() error {
	if err := i.TieredPrice.Validate(); err != nil {
		return err
	}

	if i.TieredPrice.Mode != plan.GraduatedTieredPrice {
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

	return nil
}

func splitTierRangeAtBoundary(from, to alpacadecimal.Decimal, qtyRange tierRange) []tierRange {
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
func tieredPriceCalculator(in tieredPriceCalculatorInput) error {
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

		qtyRanges = append(qtyRanges, splitTierRangeAtBoundary(in.FromQty, in.ToQty, input)...)

		previousTierQty = tierUpperBound
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
			total = total.Add(qtyRange.Tier.FlatPrice.Amount)
		}

		if qtyRange.Tier.UnitPrice != nil {
			total = total.Add(qtyRange.ToQty.Sub(qtyRange.FromQty).Mul(qtyRange.Tier.UnitPrice.Amount))
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

type newDetailedLinesInput []newDetailedLineInput

func (i newDetailedLinesInput) Sum() alpacadecimal.Decimal {
	sum := alpacadecimal.Zero

	for _, in := range i {
		sum = sum.Add(in.PerUnitAmount.Mul(in.Quantity))

		for _, discount := range in.Discounts {
			sum = sum.Sub(discount.Amount)
		}
	}

	return sum
}

type newDetailedLineInput struct {
	Name                   string                `json:"name"`
	Quantity               alpacadecimal.Decimal `json:"quantity"`
	PerUnitAmount          alpacadecimal.Decimal `json:"perUnitAmount"`
	ChildUniqueReferenceID string                `json:"childUniqueReferenceID"`
	Period                 *billingentity.Period `json:"period,omitempty"`
	// PaymentTerm is the payment term for the detailed line, defaults to arrears
	PaymentTerm plan.PaymentTermType `json:"paymentTerm,omitempty"`

	Discounts []billingentity.LineDiscount `json:"discounts,omitempty"`
}

func (i newDetailedLineInput) Validate() error {
	if i.Quantity.IsNegative() {
		return fmt.Errorf("quantity must be zero or positive")
	}

	if i.PerUnitAmount.IsNegative() {
		return fmt.Errorf("amount must be zero or positive")
	}

	if i.ChildUniqueReferenceID == "" {
		return fmt.Errorf("child unique ID is required")
	}

	if i.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

type addDiscountInput struct {
	BilledAmountBeforeLine alpacadecimal.Decimal
	MaxSpend               alpacadecimal.Decimal
}

func (i newDetailedLineInput) AddDiscountForOverage(in addDiscountInput) newDetailedLineInput {
	lineTotal := i.PerUnitAmount.Mul(i.Quantity)
	currentBillableAmount := in.BilledAmountBeforeLine.Add(lineTotal)

	if currentBillableAmount.LessThanOrEqual(in.MaxSpend) {
		// Nothing to do here
		return i
	}

	if currentBillableAmount.GreaterThanOrEqual(in.MaxSpend) && in.BilledAmountBeforeLine.GreaterThanOrEqual(in.MaxSpend) {
		// 100% discount
		i.Discounts = append(i.Discounts, billingentity.LineDiscount{
			Amount:                 lineTotal,
			Description:            formatMaximumSpendDiscountDescription(in.MaxSpend),
			ChildUniqueReferenceID: lo.ToPtr(billingentity.LineMaximumSpendReferenceID),
		})
		return i
	}

	discountAmount := currentBillableAmount.Sub(in.MaxSpend)
	i.Discounts = append(i.Discounts, billingentity.LineDiscount{
		Amount:                 discountAmount,
		Description:            formatMaximumSpendDiscountDescription(in.MaxSpend),
		ChildUniqueReferenceID: lo.ToPtr(billingentity.LineMaximumSpendReferenceID),
	})

	return i
}

func (l usageBasedLine) newDetailedLines(inputs ...newDetailedLineInput) ([]*billingentity.Line, error) {
	return slicesx.MapWithErr(inputs, func(in newDetailedLineInput) (*billingentity.Line, error) {
		if err := in.Validate(); err != nil {
			return nil, err
		}

		period := l.line.Period
		if in.Period != nil {
			period = *in.Period
		}

		return &billingentity.Line{
			LineBase: billingentity.LineBase{
				Namespace:              l.line.Namespace,
				Type:                   billingentity.InvoiceLineTypeFee,
				Status:                 billingentity.InvoiceLineStatusDetailed,
				Period:                 period,
				Name:                   in.Name,
				InvoiceAt:              l.line.InvoiceAt,
				InvoiceID:              l.line.InvoiceID,
				Currency:               l.line.Currency,
				ChildUniqueReferenceID: &in.ChildUniqueReferenceID,
				ParentLineID:           lo.ToPtr(l.line.ID),
				TaxConfig:              l.line.TaxConfig,
			},
			FlatFee: billingentity.FlatFeeLine{
				PaymentTerm:   lo.CoalesceOrEmpty(in.PaymentTerm, plan.InArrearsPaymentTerm),
				PerUnitAmount: in.PerUnitAmount,
				Quantity:      in.Quantity,
			},
			Discounts: billingentity.NewLineDiscounts(in.Discounts),
		}, nil
	})
}

func formatMaximumSpendDiscountDescription(amount alpacadecimal.Decimal) *string {
	// TODO[later]: currency formatting
	return lo.ToPtr(fmt.Sprintf("Maximum spend discount for charges over %s", amount))
}
