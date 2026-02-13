package expand

import (
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"
)

type Expandable[T any] interface {
	comparable
	Values() []T
}

type Expand[T Expandable[T]] []T

func (e Expand[T]) Validate() error {
	var errs []error

	var empty T
	values := empty.Values()

	for _, item := range e {
		if !slices.Contains(values, item) {
			errs = append(errs, fmt.Errorf("invalid expand value: %v", item))
		}
	}

	return errors.Join(errs...)
}

func (e Expand[T]) Has(value T) bool {
	return slices.Contains(e, value)
}

func (e Expand[T]) Clone() Expand[T] {
	out := make(Expand[T], len(e))
	copy(out, e)
	return out
}

func (e Expand[T]) With(value T) Expand[T] {
	cloned := e.Clone()

	if slices.Contains(cloned, value) {
		return cloned
	}

	cloned = append(cloned, value)

	return cloned
}

func (e Expand[T]) Without(value T) Expand[T] {
	return lo.Filter(e, func(item T, _ int) bool {
		return item != value
	})
}

// SetOrUnsetIf sets the value if the condition is true, otherwise it removes the value if present.
func (e Expand[T]) SetOrUnsetIf(condition bool, value T) Expand[T] {
	if condition {
		return e.With(value)
	}

	return e.Without(value)
}
