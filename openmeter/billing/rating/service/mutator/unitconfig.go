package mutator

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

// UnitConfigConversion applies the UnitConfig conversion to the metered quantity,
// transforming raw meter units into billing units (e.g., bytes to GB).
type UnitConfigConversion struct{}

var _ PreCalculationMutator = (*UnitConfigConversion)(nil)

func (m *UnitConfigConversion) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	uc := l.GetUnitConfig()
	if uc == nil {
		return l, nil
	}

	usage, err := l.GetUsage()
	if err != nil {
		return l, fmt.Errorf("getting usage for unit config conversion: %w", err)
	}

	// Apply conversion without rounding (rounding is done separately for invoicing)
	converted := uc.Convert(usage.Quantity)
	l.Usage.Quantity = converted

	l.Usage.PreLinePeriodQuantity = uc.Convert(usage.PreLinePeriodQuantity)

	return l, nil
}

// UnitConfigRounding applies rounding to the converted quantity before it reaches the pricer.
// This runs after UnitConfigConversion and before DiscountUsage.
type UnitConfigRounding struct{}

var _ PreCalculationMutator = (*UnitConfigRounding)(nil)

func (m *UnitConfigRounding) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	uc := l.GetUnitConfig()
	if uc == nil {
		return l, nil
	}

	usage, err := l.GetUsage()
	if err != nil {
		return l, fmt.Errorf("getting usage for unit config rounding: %w", err)
	}

	rounded := uc.Round(usage.Quantity)
	l.Usage.Quantity = rounded

	roundedPre := uc.Round(usage.PreLinePeriodQuantity)
	l.Usage.PreLinePeriodQuantity = roundedPre

	return l, nil
}
