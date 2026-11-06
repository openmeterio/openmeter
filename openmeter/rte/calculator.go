package rte

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Key identifies the current state of one calculation, without selecting a historical time.
type Key interface {
	models.Validator
}

type CalculateInput[K Key] struct {
	Key K

	// At is the invocation time supplied by RTE, not a historical query selector
	// or a guarantee that every source is read at the same instant.
	At time.Time
}

func (i CalculateInput[K]) Validate() error {
	var errs []error

	if err := i.Key.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.At.IsZero() {
		errs = append(errs, errors.New("calculation time is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Calculator supplies the domain result; RTE owns persistence of that result.
type Calculator[K Key, Result any] interface {
	Calculate(context.Context, CalculateInput[K]) (Result, error)
}
