package lineservice

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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

func (l *usageBasedLine) UpdateTotals() error {
	return UpdateTotalsFromDetailedLines(l.line)
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

func (l usageBasedLine) calculateDetailedLines() (newDetailedLinesInput, error) {
	pricer, err := newPricerFor(l.line)
	if err != nil {
		return nil, err
	}

	return pricer.Calculate(PricerCalculateInput(l))
}

func formatMaximumSpendDiscountDescription(amount alpacadecimal.Decimal) *string {
	// TODO[OM-1019]: currency formatting!
	return lo.ToPtr(fmt.Sprintf("Maximum spend discount for charges over %s", amount))
}

func newPricerFor(line PriceAccessor) (Pricer, error) {
	price := line.GetPrice()
	if price == nil {
		return nil, errors.New("price is nil")
	}

	if price.Type() == productcatalog.FlatPriceType {
		return &priceMutator{
			Pricer: flatPricer{},
			PostCalculation: []PostCalculationMutator{
				&discountPercentageMutator{},
				&creditsMutator{},
			},
		}, nil
	}

	var basePricer Pricer

	switch price.Type() {
	case productcatalog.UnitPriceType:
		basePricer = unitPricer{}
	case productcatalog.TieredPriceType:
		basePricer = tieredPricer{}
	case productcatalog.PackagePriceType:
		basePricer = packagePricer{}
	case productcatalog.DynamicPriceType:
		basePricer = dynamicPricer{}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", price.Type())
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
			&creditsMutator{},
		},
	}, nil
}
