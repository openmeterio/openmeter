package customerbalance

import (
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// Feature filters are represented as a tri-state option:
// - None: all credit routes / total portfolio balance.
// - Some(nil): routes without feature restrictions.
// - Some(one feature): routes without feature restrictions plus routes containing that feature.
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

	return features.ValidateAsFeatureFilter()
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

func ValidatePlanFilter(filter mo.Option[*ledger.PlanFilter]) error {
	if plan, ok := filter.Get(); ok && plan != nil {
		return plan.ValidateAsPlanFilter()
	}

	return nil
}

// creditFilterRoute defines the queried balance view, including shared-credit
// exposure on each selected dimension. Actual credit matching is checked
// separately when live impacts are applied to concrete credit sources.
func creditFilterRoute(featureFilter mo.Option[creditpurchase.FeatureFilters], planFilter mo.Option[*ledger.PlanFilter]) ledger.RouteFilter {
	route := ledger.RouteFilter{MatchPlan: planFilter}
	if featureFilter.IsAbsent() {
		return route
	}

	features := featureFilter.OrEmpty()
	if features == nil {
		route.Features = mo.Some[[]string](nil)
		return route
	}

	route.MatchFeature = features[0]
	return route
}
