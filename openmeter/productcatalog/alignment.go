package productcatalog

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/isodate"
)

type Alignment struct {
	// BillablesMustAlign indicates whether all billable items in a given phase must share the same BillingPeriodDuration.
	BillablesMustAlign bool `json:"billablesMustAlign"`
}

// AlignmentUpdate is used for the nil-ish comparison of the plan service only
type AlignmentUpdate struct {
	BillablesMustAlign *bool `json:"billablesMustAlign,omitempty"`
}

// Alignment means that either
// - the two cadences are identical
// - if a RateCard's cadence is "longer" than the Plan's cadence, the plan cadence must "divide" without remainder the ratecard's cadence
// - if a RateCard's cadence is "shorter" than the Plan's cadence, the ratecard's cadence must "divide" without remainder the plan's cadence
// "longer" and "shorter" are not generally meaningful terms for periods, as for instance sometimes P1M equals P4W, sometimes its longer.
func ValidateBillingCadencesAlign(planBillingCadence isodate.Period, rateCardBillingCadence isodate.Period) error {
	pSimple := planBillingCadence.Simplify(true)
	rcSimple := rateCardBillingCadence.Simplify(true)

	// If the two cadences are identical, we're good
	if pSimple.Equal(&rcSimple) {
		return nil
	}

	// We'll leverage the fact that Period.DibisibleBy() works correctly regardless which period is larger,
	// so we'll just test both ways

	ok, err := pSimple.DivisibleBy(rcSimple)
	if ok && err == nil {
		return nil
	}

	ok, err = rcSimple.DivisibleBy(pSimple)
	if ok && err == nil {
		return nil
	}

	return fmt.Errorf("billing cadences do not align: %s and %s", planBillingCadence, rateCardBillingCadence)
}
