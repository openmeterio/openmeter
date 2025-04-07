package lineservice

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ Line = (*usageBasedLine)(nil)

const (
	UsageChildUniqueReferenceID    = "usage"
	MinSpendChildUniqueReferenceID = "min-spend"

	// TODO[later]: Per type unique reference IDs are to be deprecated, we should use the generic names for
	// lines with one child. (e.g. graduated can stay for now, as it has multiple children)
	FlatPriceChildUniqueReferenceID = "flat-price"

	UnitPriceUsageChildUniqueReferenceID    = "unit-price-usage"
	UnitPriceMinSpendChildUniqueReferenceID = "unit-price-min-spend"
	UnitPriceMaxSpendChildUniqueReferenceID = "unit-price-max-spend"

	DynamicPriceUsageChildUniqueReferenceID = "dynamic-price-usage"

	VolumeFlatPriceChildUniqueReferenceID = "volume-flat-price"
	VolumeUnitPriceChildUniqueReferenceID = "volume-tiered-price"
	VolumeMinSpendChildUniqueReferenceID  = "volume-min-spend"

	GraduatedTieredPriceUsageChildUniqueReferenceID = "graduated-tiered-%d-price-usage"
	GraduatedTieredFlatPriceChildUniqueReferenceID  = "graduated-tiered-%d-flat-price"
	GraduatedMinSpendChildUniqueReferenceID         = "graduated-tiered-min-spend"
)

var DecimalOne = alpacadecimal.NewFromInt(1)

type usageBasedLine struct {
	lineBase
}

func (l usageBasedLine) PrepareForCreate(context.Context) (Line, error) {
	l.line.Period = l.line.Period.Truncate(billing.DefaultMeterResolution)

	return &l, nil
}

func (l usageBasedLine) Validate(ctx context.Context, targetInvoice *billing.Invoice) error {
	if _, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey); err != nil {
		return err
	}

	if err := l.lineBase.Validate(ctx, targetInvoice); err != nil {
		return err
	}

	if len(targetInvoice.Customer.UsageAttribution.SubjectKeys) == 0 {
		return billing.ValidationError{
			Err: billing.ErrInvoiceCreateUBPLineCustomerHasNoSubjects,
		}
	}

	if l.line.LineBase.Period.Truncate(billing.DefaultMeterResolution).IsEmpty() {
		return billing.ValidationError{
			Err: billing.ErrInvoiceCreateUBPLinePeriodIsEmpty,
		}
	}

	return nil
}

func (l usageBasedLine) CanBeInvoicedAsOf(ctx context.Context, in CanBeInvoicedAsOfInput) (*billing.Period, error) {
	// TODO: replace

	// TODO: validate
	if !in.ProgressiveBilling {
		// If we are not doing progressive billing, we can only bill the line if asof >= line.period.end
		if in.AsOf.Before(l.line.Period.End) {
			return nil, nil
		}

		return &l.line.Period, nil
	}

	if l.line.UsageBased.Price.Type() == productcatalog.TieredPriceType {
		tiered, err := l.line.UsageBased.Price.AsTiered()
		if err != nil {
			return nil, err
		}

		if tiered.Mode == productcatalog.VolumeTieredPrice {
			if l.line.ParentLine != nil {
				if in.AsOf.Before(l.line.ParentLine.Period.End) {
					return nil, nil
				}

				return &l.line.Period, nil
			}

			// Volume tiers are only billable if we have all the data acquired, as otherwise
			// we might overcharge the customer (if we are at tier bundaries)
			if in.AsOf.Before(l.line.Period.End) {
				return nil, nil
			}
			return &l.line.Period, nil
		}
	}

	meterAndFactory, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return nil, err
	}

	meter := meterAndFactory.meter

	asOfTruncated := in.AsOf.Truncate(billing.DefaultMeterResolution)

	switch meter.Aggregation {
	case meterpkg.MeterAggregationSum, meterpkg.MeterAggregationCount,
		meterpkg.MeterAggregationMax, meterpkg.MeterAggregationUniqueCount:

		periodStartTrucated := l.line.Period.Start.Truncate(billing.DefaultMeterResolution)

		if !periodStartTrucated.Before(asOfTruncated) {
			return nil, nil
		}

		candidatePeriod := billing.Period{
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

func (l *usageBasedLine) UpdateTotals() error {
	// Calculate the line totals
	for _, line := range l.line.Children.OrEmpty() {
		if line.DeletedAt != nil {
			continue
		}

		lineSvc, err := l.service.FromEntity(line)
		if err != nil {
			return fmt.Errorf("creating line service: %w", err)
		}

		if err := lineSvc.UpdateTotals(); err != nil {
			return fmt.Errorf("updating totals for line[%s]: %w", line.ID, err)
		}
	}

	// WARNING: Even if tempting to add discounts etc. here to the totals, we should always keep the logic as is.
	// The usageBasedLine will never be syncorinzed directly to stripe or other apps, only the detailed lines.
	//
	// Given that the external systems will have their own logic for calculating the totals, we cannot expect
	// any custom logic implemented here to be carried over to the external systems.

	// UBP line's value is the sum of all the children
	res := billing.Totals{}

	res = res.Add(lo.Map(l.line.Children.OrEmpty(), func(l *billing.Line, _ int) billing.Totals {
		// Deleted lines are not contributing to the totals
		if l.DeletedAt != nil {
			return billing.Totals{}
		}

		return l.Totals
	})...)

	l.line.LineBase.Totals = res

	return nil
}

func (l *usageBasedLine) SnapshotQuantity(ctx context.Context, invoice *billing.Invoice) error {
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
	l.line.UsageBased.MeteredQuantity = lo.ToPtr(usage.LinePeriodQty)
	l.line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)

	return nil
}

func (l *usageBasedLine) CalculateDetailedLines() error {
	if l.line.UsageBased.Quantity == nil || l.line.UsageBased.PreLinePeriodQuantity == nil {
		// This is an internal logic error, as the snapshotting should have set these values
		return fmt.Errorf("quantity and pre-line period quantity must be set for line[%s]", l.line.ID)
	}

	newDetailedLinesInput, err := l.calculateDetailedLines(&featureUsageResponse{
		LinePeriodQty:    *l.line.UsageBased.Quantity,
		PreLinePeriodQty: *l.line.UsageBased.PreLinePeriodQuantity,
	})
	if err != nil {
		return err
	}

	detailedLines, err := l.newDetailedLines(newDetailedLinesInput...)
	if err != nil {
		return fmt.Errorf("detailed lines: %w", err)
	}

	childrenWithIDReuse, err := l.line.ChildrenWithIDReuse(detailedLines)
	if err != nil {
		return fmt.Errorf("failed to reuse child IDs: %w", err)
	}

	l.line.Children = childrenWithIDReuse

	return nil
}

func (l usageBasedLine) calculateDetailedLines(ctx context.Context, usage *featureUsageResponse) (newDetailedLinesInput, error) {
	priceType := l.line.UsageBased.Price.Type()

	// Special case: flat fee is not really a usage-based line, so we handle it separately
	if priceType == productcatalog.FlatPriceType {
		flatPrice, err := l.line.UsageBased.Price.AsFlat()
		if err != nil {
			return nil, fmt.Errorf("converting price to flat price: %w", err)
		}
		return l.calculateFlatPriceDetailedLines(usage, flatPrice)
	}

	pricer, ok := pricerByPriceType[priceType]
	if !ok {
		return nil, fmt.Errorf("unsupported price type: %s", priceType)
	}

	return pricer.Calculate(ctx, l.line)
}

func (l usageBasedLine) calculateFlatPriceDetailedLines(_ *featureUsageResponse, flatPrice productcatalog.FlatPrice) (newDetailedLinesInput, error) {
	// Flat price is the same as the non-metered version, we just allow attaching entitlements to it
	switch {
	case flatPrice.PaymentTerm == productcatalog.InAdvancePaymentTerm && l.IsFirstInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InAdvancePaymentTerm,
			},
		}, nil
	case flatPrice.PaymentTerm != productcatalog.InAdvancePaymentTerm && l.IsLastInPeriod():
		return newDetailedLinesInput{
			{
				Name:                   l.line.Name,
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          flatPrice.Amount,
				ChildUniqueReferenceID: FlatPriceChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}, nil
	}

	return nil, nil
}

type applyCommitmentsInput struct {
	Commitments productcatalog.Commitments

	DetailedLines newDetailedLinesInput

	AmountBilledInPreviousPeriods alpacadecimal.Decimal

	MinimumSpendReferenceID string
}

func (i applyCommitmentsInput) Validate() error {
	if i.MinimumSpendReferenceID == "" {
		return fmt.Errorf("minimum spend reference ID is required")
	}

	return nil
}

func (l usageBasedLine) applyCommitments(in applyCommitmentsInput) (newDetailedLinesInput, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// let's add maximum spend discounts if needed

	if in.Commitments.MaximumAmount != nil {
		currentSpendAmount := in.AmountBilledInPreviousPeriods
		maxSpend := l.currency.RoundToPrecision(*in.Commitments.MaximumAmount)

		for idx, line := range in.DetailedLines {
			// Total spends after adding the line's amount
			in.DetailedLines[idx] = line.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: currentSpendAmount,
				MaxSpend:               maxSpend,
				Currency:               l.currency,
			})

			currentSpendAmount = currentSpendAmount.Add(line.TotalAmount(l.currency))
		}
	}

	if l.IsLastInPeriod() && in.Commitments.MinimumAmount != nil {
		toBeBilledAmount := in.AmountBilledInPreviousPeriods.Add(
			in.DetailedLines.Sum(l.currency),
		)

		if toBeBilledAmount.LessThan(*in.Commitments.MinimumAmount) {
			period := l.line.Period
			if l.line.ParentLine != nil {
				period = l.line.ParentLine.Period
			}

			minSpendAmount := l.currency.RoundToPrecision(in.Commitments.MinimumAmount.Sub(toBeBilledAmount))

			if minSpendAmount.IsPositive() {
				in.DetailedLines = append(in.DetailedLines, newDetailedLineInput{
					Name:          fmt.Sprintf("%s: minimum spend", l.line.Name),
					Quantity:      alpacadecimal.NewFromFloat(1),
					PerUnitAmount: minSpendAmount,
					// Minimum spend is always billed for the whole period
					Period: &period,

					ChildUniqueReferenceID: in.MinimumSpendReferenceID,
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					Category:               billing.FlatFeeCategoryCommitment,
				})
			}
		}
	}

	return in.DetailedLines, nil
}

func (l usageBasedLine) calculateUnitPriceDetailedLines(usage *featureUsageResponse, unitPrice productcatalog.UnitPrice) (newDetailedLinesInput, error) {
	var out newDetailedLinesInput

	if usage.LinePeriodQty.IsPositive() {
		out = newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               usage.LinePeriodQty,
				PerUnitAmount:          unitPrice.Amount,
				ChildUniqueReferenceID: UnitPriceUsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}
	}

	amountBilledInPreviousPeriods := l.currency.RoundToPrecision(usage.PreLinePeriodQty.Mul(unitPrice.Amount))

	detailedLines, err := l.applyCommitments(applyCommitmentsInput{
		Commitments:                   unitPrice.Commitments,
		DetailedLines:                 out,
		AmountBilledInPreviousPeriods: amountBilledInPreviousPeriods,
		MinimumSpendReferenceID:       UnitPriceMinSpendChildUniqueReferenceID,
	})
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}

func (l usageBasedLine) calculateDynamicPriceDetailedLines(usage *featureUsageResponse, dynamicPrice productcatalog.DynamicPrice) (newDetailedLinesInput, error) {
	var out newDetailedLinesInput

	if usage.LinePeriodQty.IsPositive() {
		amountInPeriod := l.currency.RoundToPrecision(
			usage.LinePeriodQty.Mul(dynamicPrice.Multiplier),
		)

		out = newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          amountInPeriod,
				ChildUniqueReferenceID: UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}
	}

	amountBilledInPreviousPeriods := l.currency.RoundToPrecision(usage.PreLinePeriodQty.Mul(dynamicPrice.Multiplier))

	detailedLines, err := l.applyCommitments(applyCommitmentsInput{
		Commitments:                   dynamicPrice.Commitments,
		DetailedLines:                 out,
		AmountBilledInPreviousPeriods: amountBilledInPreviousPeriods,
		MinimumSpendReferenceID:       MinSpendChildUniqueReferenceID,
	})
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}

func (l usageBasedLine) getNumberOfPackages(qty alpacadecimal.Decimal, packageSize alpacadecimal.Decimal) alpacadecimal.Decimal {
	requiredPackages := qty.Div(packageSize).Floor()

	if qty.Mod(packageSize).IsZero() {
		return requiredPackages
	}

	return requiredPackages.Add(DecimalOne)
}

func (l usageBasedLine) calculatePackagePriceDetailedLines(usage *featureUsageResponse, packagePrice productcatalog.PackagePrice) (newDetailedLinesInput, error) {
	var out newDetailedLinesInput

	totalUsage := usage.LinePeriodQty.Add(usage.PreLinePeriodQty)

	preLinePeriodPackages := l.getNumberOfPackages(usage.PreLinePeriodQty, packagePrice.QuantityPerPackage)
	if l.IsFirstInPeriod() {
		preLinePeriodPackages = alpacadecimal.Zero
	}

	postLinePeriodPackages := l.getNumberOfPackages(totalUsage, packagePrice.QuantityPerPackage)

	toBeBilledPackages := postLinePeriodPackages.Sub(preLinePeriodPackages)

	if !toBeBilledPackages.IsZero() {
		out = newDetailedLinesInput{
			{
				Name:                   fmt.Sprintf("%s: usage in period", l.line.Name),
				Quantity:               toBeBilledPackages,
				PerUnitAmount:          packagePrice.Amount,
				ChildUniqueReferenceID: UsageChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
			},
		}
	}

	detailedLines, err := l.applyCommitments(applyCommitmentsInput{
		Commitments:                   packagePrice.Commitments,
		DetailedLines:                 out,
		AmountBilledInPreviousPeriods: l.currency.RoundToPrecision(preLinePeriodPackages.Mul(packagePrice.Amount)),
		MinimumSpendReferenceID:       MinSpendChildUniqueReferenceID,
	})
	if err != nil {
		return nil, err
	}

	return detailedLines, nil
}

func (l usageBasedLine) calculateVolumeTieredPriceDetailedLines(usage *featureUsageResponse, price productcatalog.TieredPrice) (newDetailedLinesInput, error) {
	if !usage.PreLinePeriodQty.IsZero() {
		return nil, billing.ErrInvoiceLineVolumeSplitNotSupported
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
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		}

		if price.MaximumAmount != nil {
			line = line.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: out.Sum(l.currency),
				MaxSpend:               *price.MaximumAmount,
				Currency:               l.currency,
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
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		}

		if price.MaximumAmount != nil {
			line = line.AddDiscountForOverage(addDiscountInput{
				BilledAmountBeforeLine: out.Sum(l.currency),
				MaxSpend:               *price.MaximumAmount,
				Currency:               l.currency,
			})
		}

		out = append(out, line)
	}

	total := out.Sum(l.currency)

	if price.MinimumAmount != nil {
		normalizedMinimumAmount := l.currency.RoundToPrecision(*price.MinimumAmount)

		if total.LessThan(normalizedMinimumAmount) {
			out = append(out, newDetailedLineInput{
				Name:                   fmt.Sprintf("%s: minimum spend", l.line.Name),
				Quantity:               alpacadecimal.NewFromFloat(1),
				PerUnitAmount:          normalizedMinimumAmount.Sub(total),
				ChildUniqueReferenceID: VolumeMinSpendChildUniqueReferenceID,
				PaymentTerm:            productcatalog.InArrearsPaymentTerm,
				Category:               billing.FlatFeeCategoryCommitment,
			})
		}
	}

	return out, nil
}

func findTierForQuantity(price productcatalog.TieredPrice, quantity alpacadecimal.Decimal) (findTierForQuantityResult, error) {
	for i, tier := range price.WithSortedTiers().Tiers {
		if tier.UpToAmount == nil || quantity.LessThanOrEqual(*tier.UpToAmount) {
			return findTierForQuantityResult{
				Tier:  &price.Tiers[i],
				Index: i,
			}, nil
		}
	}

	// Technically this should not happen, as the last tier should have an upper limit of infinity
	return findTierForQuantityResult{}, fmt.Errorf("could not find tier for quantity %s: %w", quantity, billing.ErrInvoiceLineMissingOpenEndedTier)
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

	TierCallbackFn func(tierCallbackInput) error
	// TODO: might not be needed
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

type newDetailedLinesInput []newDetailedLineInput

// TODO: is this needed?
func (i newDetailedLinesInput) Sum(currency currencyx.Calculator) alpacadecimal.Decimal {
	sum := alpacadecimal.Zero

	for _, in := range i {
		sum = sum.Add(in.TotalAmount(currency))
	}

	return sum
}

type newDetailedLineInput struct {
	Name                   string                `json:"name"`
	Quantity               alpacadecimal.Decimal `json:"quantity"`
	PerUnitAmount          alpacadecimal.Decimal `json:"perUnitAmount"`
	ChildUniqueReferenceID string                `json:"childUniqueReferenceID"`
	Period                 *billing.Period       `json:"period,omitempty"`
	// PaymentTerm is the payment term for the detailed line, defaults to arrears
	PaymentTerm productcatalog.PaymentTermType `json:"paymentTerm,omitempty"`
	Category    billing.FlatFeeCategory        `json:"category,omitempty"`

	Discounts billing.LineDiscounts `json:"discounts,omitempty"`
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

func (i newDetailedLineInput) TotalAmount(currency currencyx.Calculator) alpacadecimal.Decimal {
	total := currency.RoundToPrecision(i.PerUnitAmount.Mul(i.Quantity))

	total = total.Sub(i.Discounts.SumAmount(currency))

	return total
}

type addDiscountInput struct {
	BilledAmountBeforeLine alpacadecimal.Decimal
	MaxSpend               alpacadecimal.Decimal
	Currency               currencyx.Calculator
}

func (i newDetailedLineInput) AddDiscountForOverage(in addDiscountInput) newDetailedLineInput {
	normalizedPreUsage := in.Currency.RoundToPrecision(in.BilledAmountBeforeLine)

	lineTotal := i.TotalAmount(in.Currency)

	totalBillableAmount := normalizedPreUsage.Add(lineTotal)

	normalizedMaxSpend := in.Currency.RoundToPrecision(in.MaxSpend)

	if totalBillableAmount.LessThanOrEqual(normalizedMaxSpend) {
		// Nothing to do here
		return i
	}

	if totalBillableAmount.GreaterThanOrEqual(normalizedMaxSpend) && in.BilledAmountBeforeLine.GreaterThanOrEqual(normalizedMaxSpend) {
		// 100% discount
		i.Discounts = append(i.Discounts, billing.NewLineDiscountFrom(billing.AmountLineDiscount{
			Amount: lineTotal,
			LineDiscountBase: billing.LineDiscountBase{
				Description:            formatMaximumSpendDiscountDescription(normalizedMaxSpend),
				ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
				Reason:                 billing.LineDiscountReasonMaximumSpend,
			},
		}))
		return i
	}

	discountAmount := totalBillableAmount.Sub(normalizedMaxSpend)
	i.Discounts = append(i.Discounts, billing.NewLineDiscountFrom(billing.AmountLineDiscount{
		Amount: discountAmount,
		LineDiscountBase: billing.LineDiscountBase{
			Description:            formatMaximumSpendDiscountDescription(normalizedMaxSpend),
			ChildUniqueReferenceID: lo.ToPtr(billing.LineMaximumSpendReferenceID),
			Reason:                 billing.LineDiscountReasonMaximumSpend,
		},
	}))

	return i
}

func (l usageBasedLine) newDetailedLines(inputs ...newDetailedLineInput) ([]*billing.Line, error) {
	return slicesx.MapWithErr(inputs, func(in newDetailedLineInput) (*billing.Line, error) {
		if err := in.Validate(); err != nil {
			return nil, err
		}

		period := l.line.Period
		if in.Period != nil {
			period = *in.Period
		}

		if in.Category == "" {
			in.Category = billing.FlatFeeCategoryRegular
		}

		line := &billing.Line{
			LineBase: billing.LineBase{
				Namespace:              l.line.Namespace,
				Type:                   billing.InvoiceLineTypeFee,
				Status:                 billing.InvoiceLineStatusDetailed,
				Period:                 period,
				Name:                   in.Name,
				ManagedBy:              billing.SystemManagedLine,
				InvoiceAt:              l.line.InvoiceAt,
				InvoiceID:              l.line.InvoiceID,
				Currency:               l.line.Currency,
				ChildUniqueReferenceID: &in.ChildUniqueReferenceID,
				ParentLineID:           lo.ToPtr(l.line.ID),
				TaxConfig:              l.line.TaxConfig,
			},
			FlatFee: &billing.FlatFeeLine{
				PaymentTerm:   lo.CoalesceOrEmpty(in.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				PerUnitAmount: in.PerUnitAmount,
				Quantity:      in.Quantity,
				Category:      in.Category,
			},
			Discounts: in.Discounts,
		}

		if err := line.Validate(); err != nil {
			return nil, err
		}

		return line, nil
	})
}

func formatMaximumSpendDiscountDescription(amount alpacadecimal.Decimal) *string {
	// TODO[OM-1019]: currency formatting!
	return lo.ToPtr(fmt.Sprintf("Maximum spend discount for charges over %s", amount))
}
