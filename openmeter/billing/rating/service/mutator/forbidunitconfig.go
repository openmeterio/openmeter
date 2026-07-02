package mutator

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

// ForbidUnitConfig is the flag-off counterpart to UnitConfig: it is registered in
// the pre-calculation pipeline when unitConfig.enabled is false. For the common case
// (no unit_config on the line) it is a no-op, but it errors when a line does carry a
// unit_config, so a config that reaches rating while the feature is disabled surfaces
// as a hard failure instead of silently billing the raw metered quantity.
type ForbidUnitConfig struct{}

var _ PreCalculationMutator = (*ForbidUnitConfig)(nil)

func (m *ForbidUnitConfig) Mutate(l rate.PricerCalculateInput) (rate.PricerCalculateInput, error) {
	if l.GetUnitConfig() != nil {
		return l, fmt.Errorf("refusing to bill raw quantity: %w", ErrUnitConfigDisabled)
	}

	return l, nil
}
