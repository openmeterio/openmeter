package statelessx

import (
	"context"
	"errors"
)

type ActionFn func(context.Context) error

// allOf chains multiple action functions into a single action function, all functions
// will be called, regardless of their error state.
// The reported errors will be joined into a single error object.
func AllOf(fn ...ActionFn) ActionFn {
	return func(ctx context.Context) error {
		var outErr error

		for _, f := range fn {
			if err := f(ctx); err != nil {
				outErr = errors.Join(outErr, err)
			}
		}

		return outErr
	}
}
