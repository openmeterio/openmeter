package rte

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type State[Result any] struct {
	// CalculatedAt orders successful attempts by their start time. It is not
	// a source revision, and ordering across replicas relies on their clocks.
	CalculatedAt time.Time `json:"calculated_at"`
	Value        Result    `json:"value"`
}

type StoreInput[K Key, Result any] struct {
	Key   K
	State State[Result]
}

func (i StoreInput[K, Result]) Validate() error {
	var errs []error

	if err := i.Key.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.State.CalculatedAt.IsZero() {
		errs = append(errs, errors.New("calculated at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Store[K Key, Result any] interface {
	// Store atomically compares CalculatedAt across writers and replaces state
	// only for a strictly later start. Equal or older attempts return the
	// retained success; callers must use this returned state, not their input.
	Store(context.Context, StoreInput[K, Result]) (State[Result], error)

	// Get returns the stored success, or NewNotCalculatedError when none exists.
	Get(context.Context, K) (State[Result], error)
}

func NewNotCalculatedError() error {
	return models.NewGenericNotFoundError(errors.New("result has not been calculated"))
}
