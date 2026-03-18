package price

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type VolumeTiered struct {
	NonProgressiveBillingPricer
}

var _ Pricer = (*VolumeTiered)(nil)

func (p VolumeTiered) GenerateDetailedLines(l PricerCalculateInput) (pricer.DetailedLines, error) {
	price, err := l.GetPrice().AsTiered()
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

	out := make(pricer.DetailedLines, 0, 2)

	findTierRes, err := p.FindTierForQuantity(price, usage.Quantity)
	if err != nil {
		return nil, err
	}

	tier := findTierRes.Tier
	tierIndex := findTierRes.Index

	if tier.FlatPrice != nil {
		line := pricer.DetailedLine{
			Name:                   fmt.Sprintf("%s: flat price for tier %d", l.GetName(), tierIndex+1),
			Quantity:               alpacadecimal.NewFromFloat(1),
			PerUnitAmount:          tier.FlatPrice.Amount,
			ChildUniqueReferenceID: pricer.VolumeFlatPriceChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		}

		out = append(out, line)
	}

	if tier.UnitPrice != nil && !usage.Quantity.IsZero() {
		line := pricer.DetailedLine{
			Name:                   fmt.Sprintf("%s: unit price for tier %d", l.GetName(), tierIndex+1),
			Quantity:               usage.Quantity,
			PerUnitAmount:          tier.UnitPrice.Amount,
			ChildUniqueReferenceID: pricer.VolumeUnitPriceChildUniqueReferenceID,
			PaymentTerm:            productcatalog.InArrearsPaymentTerm,
		}

		out = append(out, line)
	}

	return out, nil
}

type FindTierForQuantityResult struct {
	Tier  *productcatalog.PriceTier
	Index int
}

func (p VolumeTiered) FindTierForQuantity(price productcatalog.TieredPrice, quantity alpacadecimal.Decimal) (FindTierForQuantityResult, error) {
	for i, tier := range price.WithSortedTiers().Tiers {
		if tier.UpToAmount == nil || quantity.LessThanOrEqual(*tier.UpToAmount) {
			return FindTierForQuantityResult{
				Tier:  &price.Tiers[i],
				Index: i,
			}, nil
		}
	}

	// Technically this should not happen, as the last tier should have an upper limit of infinity
	return FindTierForQuantityResult{}, fmt.Errorf("could not find tier for quantity %s: %w", quantity, billing.ErrInvoiceLineMissingOpenEndedTier)
}
