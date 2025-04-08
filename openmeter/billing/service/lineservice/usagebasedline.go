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
	UnitPriceMaxSpendChildUniqueReferenceID = "unit-price-max-spend"

	DynamicPriceUsageChildUniqueReferenceID = "dynamic-price-usage"

	VolumeFlatPriceChildUniqueReferenceID = "volume-flat-price"
	VolumeUnitPriceChildUniqueReferenceID = "volume-tiered-price"

	GraduatedTieredPriceUsageChildUniqueReferenceID = "graduated-tiered-%d-price-usage"
	GraduatedTieredFlatPriceChildUniqueReferenceID  = "graduated-tiered-%d-flat-price"

	// TODO: Let's add migrations to fix the child uniqe reference ids
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
	l.line.UsageBased.PreLinePeriodQuantity = lo.ToPtr(usage.PreLinePeriodQty)

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

func (l usageBasedLine) asPricerCalculateInput() (PricerCalculateInput, error) {
	if l.line.UsageBased.Quantity == nil || l.line.UsageBased.PreLinePeriodQuantity == nil {
		return PricerCalculateInput{}, fmt.Errorf("quantity and pre-line period quantity must be set for line[%s]", l.line.ID)
	}

	return PricerCalculateInput{
		usageBasedLine:   l,
		preLinePeriodQty: *l.line.UsageBased.PreLinePeriodQuantity,
		linePeriodQty:    *l.line.UsageBased.Quantity,
	}, nil
}

func (l usageBasedLine) getPricer() (Pricer, error) {
	var basePricer Pricer

	switch l.line.UsageBased.Price.Type() {
	case productcatalog.FlatPriceType:
		basePricer = flatPricer{}
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
		Pricer: basePricer,
		PostCalculation: []PostCalculationMutator{
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

	pricerInput, err := l.asPricerCalculateInput()
	if err != nil {
		return nil, err
	}

	return pricer.Calculate(pricerInput)
}

type newDetailedLinesInput []newDetailedLineInput

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
