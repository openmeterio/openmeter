package subscription

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanRef struct {
	Id      string `json:"id"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}

func (p PlanRef) GetPath() SpecPath {
	return SpecPath(fmt.Sprintf("%s/%d", p.Key, p.Version))
}

func (p PlanRef) Equal(p2 PlanRef) bool {
	if p.Id != p2.Id {
		return false
	}
	if p.Key != p2.Key {
		return false
	}
	if p.Version != p2.Version {
		return false
	}
	return true
}

func (p *PlanRef) NilEqual(p2 *PlanRef) bool {
	if p == nil && p2 == nil {
		return true
	}
	if p != nil && p2 != nil {
		return p.Equal(*p2)
	}

	return false
}

// All methods are expected to return stable values.
type PlanRateCard interface {
	ToCreateSubscriptionItemPlanInput() CreateSubscriptionItemPlanInput
	GetKey() string
}

// All methods are expected to return stable values.
type PlanPhase interface {
	ToCreateSubscriptionPhasePlanInput() CreateSubscriptionPhasePlanInput
	GetRateCards() []PlanRateCard
	GetKey() string
}

// All methods are expected to return stable values.
type Plan interface {
	ToCreateSubscriptionPlanInput() CreateSubscriptionPlanInput

	GetName() string

	// Phases are expected to be returned in the order they activate.
	GetPhases() []PlanPhase

	// Will not make sense on the long term
	Currency() currencyx.Code
}

// NewPlanNotFoundError returns a new PlanNotFoundError.
func NewPlanNotFoundError(key string, version int) error {
	return &PlanNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("plan %s with version %d not found", key, version),
		),
	}
}

var _ models.GenericError = &PlanNotFoundError{}

// PlanNotFoundError is returned when a meter is not found.
type PlanNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *PlanNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *PlanNotFoundError) Unwrap() error {
	return e.err
}

// IsPlanNotFoundError returns true if the error is a PlanNotFoundError.
func IsPlanNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *PlanNotFoundError

	return errors.As(err, &e)
}
