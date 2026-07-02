package mutator

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// UnitConfig is a pre-calculation mutator that applies the rate card's unit_config
// conversion to the raw metered quantity before the pricer runs, so the downstream
// pricer sees billing units (e.g. GB) instead of raw metered units (e.g. bytes)
// without ever knowing a conversion happened.
//
// It converts the current-line quantity and the pre-line-period quantity
// independently, each through UnitConfig.Apply, and writes back the rounded
// (invoiced) result. Both are converted so tiered/graduated pricers evaluate their
// tier boundaries in converted units. It deliberately does NOT reconstruct a
// cumulative total as pre-line-period + current: that identity does not hold for
// every meter aggregation (e.g. MAX/UNIQUE_COUNT) or for the charges path (where
// pre-line-period is always zero), so making split lines sum correctly under
// non-linear rounding is left to the invoice-line layer where the raw metered pair
// is available.
//
// Idempotent: it only rewrites the in-memory Usage (rebuilt from the raw metered
// quantity on every run), never a persisted field, so re-rating reconverts from raw
// instead of double-converting.
type UnitConfig struct{}

var _ PreCalculationMutator = (*UnitConfig)(nil)

func (m *UnitConfig) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	unitConfig := l.GetUnitConfig()
	if unitConfig == nil {
		return l, nil
	}

	// The authoring validator forbids a unit_config on price types that cannot
	// convert (flat/package/dynamic), so reaching the mutator with an unsupported
	// price means inconsistent data. Surface it as an error rather than silently
	// billing the raw quantity — a dropped conversion would under/over-bill.
	price := l.GetPrice()
	if price == nil {
		return l, fmt.Errorf("line has no price: %w", ErrUnitConfigUnsupportedPrice)
	}
	if !price.SupportsUnitConfig() {
		return l, fmt.Errorf("price type %q: %w", price.Type(), ErrUnitConfigUnsupportedPrice)
	}

	usage, err := l.GetUsage()
	if err != nil {
		return l, fmt.Errorf("getting usage: %w", err)
	}

	usage = ApplyUnitConfig(usage, unitConfig)
	l.Usage = &usage

	return l, nil
}

// ApplyUnitConfig converts the billable quantities of usage through the unit_config,
// applying the conversion (and its rounding) INDEPENDENTLY to Quantity and
// PreLinePeriodQuantity. It is the shared contract used by both the rating
// PreCalculation mutator and the charges line-mapper, so the priced amount and the
// displayed billable quantity convert through identical logic and cannot drift. A nil
// unitConfig is the identity.
func ApplyUnitConfig(usage rating.Usage, unitConfig *productcatalog.UnitConfig) rating.Usage {
	if unitConfig == nil {
		return usage
	}

	_, usage.Quantity = unitConfig.Apply(usage.Quantity)
	_, usage.PreLinePeriodQuantity = unitConfig.Apply(usage.PreLinePeriodQuantity)

	return usage
}
