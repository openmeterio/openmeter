package statelessx

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
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

func WithParameters[T models.Validator](fn func(context.Context, T) error) func(context.Context, ...any) error {
	return func(ctx context.Context, args ...any) error {
		if len(args) == 0 {
			return fmt.Errorf("no arguments provided: expected %T", new(T))
		}

		converted, ok := args[0].(T)
		if !ok {
			return fmt.Errorf("argument %T is not %T", args[0], new(T))
		}

		if err := converted.Validate(); err != nil {
			return fmt.Errorf("validate: %w", err)
		}

		return fn(ctx, converted)
	}
}
