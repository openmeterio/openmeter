package subscription

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type PlanRefInput struct {
	Key     string `json:"key"`
	Version *int   `json:"version,omitempty"`
}

type PlanRef struct {
	Id      string `json:"id"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}

func (p PlanRef) Equals(p2 PlanRef) bool {
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

type PlanAdapter interface {
	// GetPlan returns the plan with the given key and version with all it's dependent resources.
	//
	// If the Plan is Not Found, it should return a PlanNotFoundError.
	GetVersion(ctx context.Context, namespace string, ref PlanRefInput) (Plan, error)
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

	GetRef() PlanRef

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
