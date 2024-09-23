// All contents here are dummy implementations for things not yet implemented.
package dummy

import "context"

// Dummy transaction
func Transaction[R any](ctx context.Context, fn func(ctx context.Context) (R, error)) (R, error) {
	return fn(ctx)
}
