package syncx

import (
	"context"
	"sync"
)

// OnceValues executes a function at most once and returns the cached result.
// This function behaves like sync.OnceValues, including caching both return
// values from the first call, but with a context argument to the function.
//
// Can be used to do lazy database lookups that may be needed by multiple callbacks but should execute at most once.
func OnceValues[T1, T2 any](fn func(context.Context) (T1, T2)) func(context.Context) (T1, T2) {
	var (
		once sync.Once
		v1   T1
		v2   T2
	)

	return func(ctx context.Context) (T1, T2) {
		once.Do(func() {
			v1, v2 = fn(ctx)
		})

		return v1, v2
	}
}
