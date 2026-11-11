// Package calculator delegates RTE calculations to the metered entitlement
// engine. It forwards the namespaced entitlement ID and calculation start to
// GetEntitlementBalance. The engine chooses the usage period, rounds partial
// minutes up, and applies its grant and snapshot rules.
//
// A recalculation after late usage follows the same snapshot history as a
// direct domain call. Queuing another calculation does not rebuild that history.
package calculator
