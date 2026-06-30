package mutator

import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

// UnitConfig is a pre-calculation mutator that applies the rate card's unit_config
// conversion to the raw metered quantity before the pricer runs. It converts and
// rounds on the cumulative endpoints (start = pre-line-period quantity, end = pre +
// current) and writes the rounded billed quantity back as the per-line delta, so the
// downstream pricer sees converted units without ever knowing a conversion happened.
//
// Rounding is applied on the cumulative boundary rather than per line on purpose:
// rounding is non-linear, so rounding each line independently would double-bill
// across progressive splits (⌈1.4⌉ + ⌈1.3⌉ = 4 vs the correct ⌈2.7⌉ = 3). Computing
// the diff here also matters because the Unit pricer ignores PreLinePeriodQuantity.
//
// Unlike DiscountUsage this mutator stays idempotent: it only rewrites the in-memory
// Usage (rebuilt from the raw metered quantity on every run), not a persisted field,
// so re-rating reconverts from raw instead of double-converting.
type UnitConfig struct{}

var _ PreCalculationMutator = (*UnitConfig)(nil)

func (m *UnitConfig) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	unitConfig := l.GetUnitConfig()
	if unitConfig == nil {
		return l, nil
	}

	// Defensive: the authoring validator forbids a unit_config on price types that
	// cannot convert (flat/package/dynamic), so reaching the mutator with an
	// unsupported price means inconsistent data. Skip conversion rather than
	// mis-price, leaving the pricer to bill the raw quantity as it does today.
	price := l.GetPrice()
	if price == nil || !price.SupportsUnitConfig() {
		return l, nil
	}

	usage, err := l.GetUsage()
	if err != nil {
		return l, err
	}

	_, start := unitConfig.Apply(usage.PreLinePeriodQuantity)
	_, end := unitConfig.Apply(usage.PreLinePeriodQuantity.Add(usage.Quantity))

	usage.PreLinePeriodQuantity = start
	usage.Quantity = end.Sub(start)

	l.Usage = &usage

	return l, nil
}
