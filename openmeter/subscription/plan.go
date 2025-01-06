package subscription

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type PlanRef struct {
	Id      string `json:"id"`
	Key     string `json:"key"`
	Version int    `json:"version"`
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

	// Phases are expected to be returned in the order they activate.
	GetPhases() []PlanPhase

	// Will not make sense on the long term
	Currency() currencyx.Code
}

type PlanNotFoundError struct {
	Key     string
	Version int
}

func (e PlanNotFoundError) Error() string {
	return fmt.Sprintf("plan %s@%d not found", e.Key, e.Version)
}

func (e PlanNotFoundError) Traits() []errorsx.Trait {
	return []errorsx.Trait{errorsx.NotFound}
}

var _ errorsx.ErrorWithTraits = PlanNotFoundError{}
