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
	price, err := l.line.UsageBased.Price.AsTiered()
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

	if !l.preLinePeriodQty.IsZero() {
		// Should not happen: we already somehow enabled progressive billing for volume tiered pricing
		return nil, billing.ErrInvoiceLineVolumeSplitNotSupported
	}

	out := make(newDetailedLinesInput, 0, 4)

	findTierRes, err := p.findTierForQuantity(price, l.linePeriodQty)
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

	if tier.UnitPrice != nil && !l.linePeriodQty.IsZero() {
		line := newDetailedLineInput{
			Name:                   fmt.Sprintf("%s: unit price for tier %d", l.line.Name, tierIndex+1),
			Quantity:               l.linePeriodQty,
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
