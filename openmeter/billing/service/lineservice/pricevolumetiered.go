package lineservice

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type volumeTieredPricer struct {
	NonProgressiveBillingPricer
}

var _ Pricer = (*volumeTieredPricer)(nil)

func (p volumeTieredPricer) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	price, err := l.line.Price.AsTiered()
	if err != nil {
		return nil, fmt.Errorf("converting price to volume tiered price: %w", err)
	}

	if price.Mode != productcatalog.VolumeTieredPrice {
		return nil, fmt.Errorf("price is not a volume tiered price")
	}

	if !l.IsLastInPeriod() {
		// No support for progressive billing for volume tiered pricing
		return nil, nil
	}

	usage, err := l.GetUsage()
	if err != nil {
		return nil, err
	}

	if !usage.PreLinePeriodQuantity.IsZero() {
		// Should not happen: we already somehow enabled progressive billing for volume tiered pricing
		return nil, billing.ErrInvoiceLineVolumeSplitNotSupported
	}

	out := make(newDetailedLinesInput, 0, 2)

	findTierRes, err := p.findTierForQuantity(price, usage.LinePeriodQuantity)
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

		out = append(out, line)
	}

	if tier.UnitPrice != nil && !usage.LinePeriodQuantity.IsZero() {
		line := newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: unit price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               usage.LinePeriodQuantity,
			PerUnitAmount:          tier.UnitPrice.Amount,
			ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		}

		out = append(out, line)
	}

	return out, nil
}

type findTierForQuantityResult struct {
	Tier  *productcatalog.PriceTier
	Index int
}

func (p volumeTieredPricer) findTierForQuantity(price productcatalog.TieredPrice, quantity alpacadecimal.Decimal) (findTierForQuantityResult, error) {
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
