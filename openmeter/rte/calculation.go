package rte

import "context"

// Calculation exposes the current result for a registered calculator.
type Calculation[K Key, Result any] interface {
	// Calculate runs the calculator and stores its successful result. If another
	// attempt already stored an equal or later start, it returns that retained
	// state instead. A failed calculation leaves stored state untouched.
	Calculate(context.Context, K) (State[Result], error)

	// Get only reads stored state. It never invokes the calculator, and returns
	// a not-calculated error when no successful result has been stored.
	Get(context.Context, K) (State[Result], error)
}
