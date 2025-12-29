package lineservice

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

var _ Line = (*usageBasedLine)(nil)

const (
	UsageChildUniqueReferenceID    = "usage"
	MinSpendChildUniqueReferenceID = "min-spend"

	// TODO[later]: Per type unique reference IDs are to be deprecated, we should use the generic names for
	// lines with one child. (e.g. graduated can stay for now, as it has multiple children)
	FlatPriceChildUniqueReferenceID = "flat-price"

	UnitPriceUsageChildUniqueReferenceID    = "unit-price-usage"
	UnitPriceMaxSpendChildUniqueReferenceID = "unit-price-max-spend"

	DynamicPriceUsageChildUniqueReferenceID = "dynamic-price-usage"

	VolumeFlatPriceChildUniqueReferenceID = "volume-flat-price"
	VolumeUnitPriceChildUniqueReferenceID = "volume-tiered-price"

	GraduatedTieredPriceUsageChildUniqueReferenceID = "graduated-tiered-%d-price-usage"
	GraduatedTieredFlatPriceChildUniqueReferenceID  = "graduated-tiered-%d-flat-price"

	RateCardDiscountChildUniqueReferenceID = "rateCardDiscount/correlationID=%s"
)

var DecimalOne = alpacadecimal.NewFromInt(1)

type usageBasedLine struct {
	lineBase
}

func (l usageBasedLine) PrepareForCreate(context.Context) (Line, error) {
	l.line.Period = l.line.Period.Truncate(streaming.MinimumWindowSizeDuration)
	l.line.InvoiceAt = l.line.InvoiceAt.Truncate(streaming.MinimumWindowSizeDuration)

	return &l, nil
}

func (l usageBasedLine) Validate(ctx context.Context, targetInvoice *billing.Invoice) error {
	if _, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey); err != nil {
		return err
	}

	if err := l.lineBase.Validate(ctx, targetInvoice); err != nil {
		return err
	}

	if l.line.LineBase.Period.Truncate(streaming.MinimumWindowSizeDuration).IsEmpty() {
		return billing.ValidationError{
			Err: billing.ErrInvoiceCreateUBPLinePeriodIsEmpty,
		}
	}

	return nil
}

func (l usageBasedLine) CanBeInvoicedAsOf(ctx context.Context, in CanBeInvoicedAsOfInput) (*billing.Period, error) {
	if !in.ProgressiveBilling {
		// If we are not doing progressive billing, we can only bill the line if asof >= line.period.end
		if in.AsOf.Before(l.line.Period.End) {
			return nil, nil
		}

		return &l.line.Period, nil
	}

	// Progressive billing checks
	pricer, err := l.getPricer()
	if err != nil {
		return nil, err
	}

	canBeInvoiced, err := pricer.CanBeInvoicedAsOf(l, in.AsOf)
	if err != nil {
		return nil, err
	}

	if !canBeInvoiced {
		// If the pricer cannot be invoiced most probably due to the missing progressive billing support
		// or invalid input, we should not bill the line
		return nil, nil
	}

	// Let's check if the underlying meter can be billed in a progressive manner
	meterAndFactory, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return nil, err
	}

	meter := meterAndFactory.meter

	asOfTruncated := in.AsOf.Truncate(streaming.MinimumWindowSizeDuration)

	switch meter.Aggregation {
	case meterpkg.MeterAggregationSum, meterpkg.MeterAggregationCount,
		meterpkg.MeterAggregationMax, meterpkg.MeterAggregationUniqueCount:

		periodStartTrucated := l.line.Period.Start.Truncate(streaming.MinimumWindowSizeDuration)

		if !periodStartTrucated.Before(asOfTruncated) {
			return nil, nil
		}

		candidatePeriod := billing.Period{
			Start: periodStartTrucated,
			End:   asOfTruncated,
		}

		if candidatePeriod.End.After(l.line.Period.End) {
			candidatePeriod.End = l.line.Period.End
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
	return l.service.UpdateTotalsFromDetailedLines(l.line)
}

func (l *usageBasedLine) SnapshotQuantity(ctx context.Context, customer billing.InvoiceCustomer) error {
	featureMeter, err := l.service.resolveFeatureMeter(ctx, l.line.Namespace, l.line.UsageBased.FeatureKey)
	if err != nil {
		return err
	}

	usage, err := l.service.getFeatureUsage(ctx,
		getFeatureUsageInput{
			Line:     l.line,
			Feature:  featureMeter.feature,
			Meter:    featureMeter.meter,
			Customer: customer,
		},
	)
	if err != nil {
		return err
	}

	// MeteredQuantity is not mutable by the price mutators, that's why we have this redundancy
	l.line.UsageBased.MeteredQuantity = lo.ToPtr(usage.LinePeriodQty)
	l.line.UsageBased.Quantity = lo.ToPtr(usage.LinePeriodQty)
	l.line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)
	l.line.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)
	return nil
}

func (l *usageBasedLine) CalculateDetailedLines() error {
	if l.line.UsageBased.Quantity == nil || l.line.UsageBased.PreLinePeriodQuantity == nil {
		// This is an internal logic error, as the snapshotting should have set these values
		return fmt.Errorf("quantity and pre-line period quantity must be set for line[%s]", l.line.ID)
	}

	newDetailedLinesInput, err := l.calculateDetailedLines()
	if err != nil {
		return err
	}

	if err := mergeDetailedLines(l.line, newDetailedLinesInput); err != nil {
		return fmt.Errorf("merging detailed lines: %w", err)
	}

	return nil
}

func (l usageBasedLine) getPricer() (Pricer, error) {
	var basePricer Pricer

	switch l.line.UsageBased.Price.Type() {
	case productcatalog.UnitPriceType:
		basePricer = unitPricer{}
	case productcatalog.TieredPriceType:
		basePricer = tieredPricer{}
	case productcatalog.PackagePriceType:
		basePricer = packagePricer{}
	case productcatalog.DynamicPriceType:
		basePricer = dynamicPricer{}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", l.line.UsageBased.Price.Type())
	}

	// This priceMutator captures the calculation flow for discounts and commitments:
	return &priceMutator{
		PreCalculation: []PreCalculationMutator{
			&setQuantityToMeteredQuantity{},
			&discountUsageMutator{},
		},
		Pricer: basePricer,
		PostCalculation: []PostCalculationMutator{
			&discountPercentageMutator{},
			&maxAmountCommitmentMutator{},
			&minAmountCommitmentMutator{},
		},
	}, nil
}

func (l usageBasedLine) calculateDetailedLines() (newDetailedLinesInput, error) {
	pricer, err := l.getPricer()
	if err != nil {
		return nil, err
	}

	return pricer.Calculate(PricerCalculateInput(l))
}

func formatMaximumSpendDiscountDescription(amount alpacadecimal.Decimal) *string {
	// TODO[OM-1019]: currency formatting!
	return lo.ToPtr(fmt.Sprintf("Maximum spend discount for charges over %s", amount))
}

func (l usageBasedLine) IsPeriodEmptyConsideringTruncations() bool {
	return l.Period().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty()
}
