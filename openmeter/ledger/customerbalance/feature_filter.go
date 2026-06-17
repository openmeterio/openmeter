package customerbalance

import (
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// Feature filters are represented as a tri-state option:
// - None: all credit routes / total portfolio balance.
// - Some(nil): unrestricted routes only.
// - Some(one feature): unrestricted routes plus restricted routes containing that feature.
func AllFeatureFilter() mo.Option[creditpurchase.FeatureFilters] {
	return mo.None[creditpurchase.FeatureFilters]()
}

func NewFeatureFilter(features []string) mo.Option[creditpurchase.FeatureFilters] {
	return mo.Some(creditpurchase.FeatureFilters(features).Normalize())
}

func NewUnrestrictedFeatureFilter() mo.Option[creditpurchase.FeatureFilters] {
	return mo.Some[creditpurchase.FeatureFilters](nil)
}

func ValidateFeatureFilter(filter mo.Option[creditpurchase.FeatureFilters]) error {
	if filter.IsAbsent() {
		return nil
	}

	features := filter.OrEmpty()
	if features == nil {
		return nil
	}

	switch len(features) {
	case 0:
		return errors.New("features are required when feature filter is restricted")
	case 1:
	default:
		return errors.New("feature-filtered balance supports exactly one feature")
	}

	if err := features.Validate(); err != nil {
		return fmt.Errorf("features: %w", err)
	}

	return nil
}

func normalizeFeatureFilter(filter mo.Option[creditpurchase.FeatureFilters]) mo.Option[creditpurchase.FeatureFilters] {
	if filter.IsAbsent() {
		return filter
	}

	features := filter.OrEmpty()
	if features == nil {
		return filter
	}

	return mo.Some(features.Normalize())
}

func featureFilterRoute(featureFilter mo.Option[creditpurchase.FeatureFilters]) ledger.RouteFilter {
	if featureFilter.IsAbsent() {
		return ledger.RouteFilter{}
	}

	features := featureFilter.OrEmpty()
	if features == nil {
		return ledger.RouteFilter{Features: mo.Some[[]string](nil)}
	}

	return ledger.RouteFilter{MatchFeature: features[0]}
}
