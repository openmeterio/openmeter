package optimizedengine

import (
	"context"
	"time"
)

// engine.Engine only has a single dependency provided, a QueryUsageFn.
// All optimizations work by optimizing the underlying implementation of this QueryUsageFn
// (usually done by pre-fetching data and caching it)
// Guards are present to determine if a given optimization can successfully be used for the query being made (e.g. will it result in a cache miss)
// If a given guard fails, the invocation CAN decide to fallback to a less efficient implementation
type OptimizationGuard func(ctx context.Context, from, to time.Time) error
