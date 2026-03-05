package cost

import "context"

// Adapter provides the data access layer for cost computation.
type Adapter interface {
	QueryFeatureCost(ctx context.Context, input QueryFeatureCostInput) (*CostQueryResult, error)
}
