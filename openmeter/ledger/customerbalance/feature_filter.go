package customerbalance

import (
	"slices"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// Feature filters are represented as a tri-state option:
// - None: all credit routes / total portfolio balance.
// - Some(nil): unrestricted routes only.
// - Some(features): unrestricted routes plus restricted routes containing any of those features.
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

// featureFilterRoute maps the filter to the feature dimension of a
// ledger.RouteFilter. RouteFilter.MatchFeature is singular, so a multi-feature
// filter cannot be pushed into the route query: for that shape this returns an
// empty feature dimension and callers must match routes in Go via
// routeMatchesAnyFeature (see multiFeatureKeys).
func featureFilterRoute(featureFilter mo.Option[creditpurchase.FeatureFilters]) ledger.RouteFilter {
	if featureFilter.IsAbsent() {
		return ledger.RouteFilter{}
	}

	features := featureFilter.OrEmpty()
	if features == nil {
		return ledger.RouteFilter{Features: mo.Some[[]string](nil)}
	}

	if len(features) > 1 {
		return ledger.RouteFilter{}
	}

	return ledger.RouteFilter{MatchFeature: features[0]}
}

// multiFeatureKeys returns the requested feature keys when the filter names
// more than one feature — the shape ledger.RouteFilter cannot express, so
// balance reads must enumerate sub-accounts and match their routes in Go
// instead of aggregating with a route query.
func multiFeatureKeys(featureFilter mo.Option[creditpurchase.FeatureFilters]) ([]string, bool) {
	if featureFilter.IsAbsent() {
		return nil, false
	}

	features := featureFilter.OrEmpty()
	if len(features) <= 1 {
		return nil, false
	}

	return features, true
}

// routeMatchesAnyFeature mirrors ledger.RouteFilter.MatchFeature's coverage
// semantics for a multi-key filter: unrestricted routes always match (shared
// credit is spendable by any feature), restricted routes match when their
// restriction includes any of the requested keys.
func routeMatchesAnyFeature(route ledger.Route, keys []string) bool {
	if len(route.Features) == 0 {
		return true
	}

	return slices.ContainsFunc(keys, func(key string) bool {
		return slices.Contains(route.Features, key)
	})
}
