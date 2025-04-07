package lineservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

var _ Pricer = (*volumeTieredPricer)(nil)

type volumeTieredPricer struct{}

func (p *volumeTieredPricer) Calculate(ctx context.Context, l usageBasedLine) (newDetailedLinesInput, error) {
	price, err := l.line.UsageBased.Price.AsTiered()
	if err != nil {
		return nil, err
	}

	if price.Mode != productcatalog.VolumeTieredPrice {
		return nil, errors.New("price is not a volume tiered price")
	}

	if l.line.UsageBased.Quantity == nil {
		return nil, errors.New("usage based line has no quantity")
	}

	// Let's validate that we are not progressively billing this line, by
	// Only billing the last line in the period.
	if !l.IsLastInPeriod() {
		return nil, nil
	}

	// make sure we are not creating a split line detailed line.
	if l.line.UsageBased.PreLinePeriodQuantity != nil && l.line.UsageBased.PreLinePeriodQuantity.IsPositive() {
		return nil, billing.ErrInvoiceLineVolumeSplitNotSupported
	}

	linePeriodQty := *l.line.UsageBased.Quantity

	out := make(newDetailedLinesInput, 0, 2)

	findTierRes, err := p.findTierForQuantity(price, linePeriodQty)
	if err != nil {
		return nil, err
	}

	tier := findTierRes.Tier
	tierIndex := findTierRes.Index

	if tier.FlatPrice != nil {
		out = append(out, newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: flat price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          tier.FlatPrice.Amount,
			ChildUniqueReferenceID: VolumeFlatPriceChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		})
	}

	if tier.UnitPrice != nil && !linePeriodQty.IsZero() {
		out = append(out, newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: unit price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               linePeriodQty,
			PerUnitAmount:          tier.UnitPrice.Amount,
			ChildUniqueReferenceID: VolumeUnitPriceChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		})
	}

	return out, nil
}

type findTierForQuantityResult struct {
	Tier  *productcatalog.PriceTier
	Index int
}

func (p *volumeTieredPricer) findTierForQuantity(price productcatalog.TieredPrice, quantity alpacadecimal.Decimal) (findTierForQuantityResult, error) {
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

func (p *volumeTieredPricer) Capabilities(l usageBasedLine) (PricerCapabilities, error) {
	return PricerCapabilities{
		AllowsProgressiveBilling: false,
	}, nil
}
