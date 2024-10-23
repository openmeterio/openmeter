package subscription

import (
	"context"
	"fmt"
)

type PlanAdapter interface {
	// GetPlan returns the plan with the given key and version with all it's dependent resources.
	//
	// If the Plan is Not Found, it should return a PlanNotFoundError.
	GetVersion(ctx context.Context, planKey string, version int) (Plan, error)
}

// All methods are expected to return stable values.
type RateCard interface {
	ToCreateSubscriptionItemPlanInput() CreateSubscriptionItemPlanInput
	GetKey() string
}

// All methods are expected to return stable values.
type PlanPhase interface {
	ToCreateSubscriptionPhasePlanInput() CreateSubscriptionPhasePlanInput
	GetRateCards() []RateCard
	GetKey() string
}

// All methods are expected to return stable values.
type Plan interface {
	ToCreateSubscriptionPlanInput() CreateSubscriptionPlanInput
	// Phases are expected to be returned in the order they activate.
	GetPhases() []PlanPhase
	GetKey() string
	GetVersionNumber() int
}

type PlanNotFoundError struct {
	Key     string
	Version int
}

func (e *PlanNotFoundError) Error() string {
	return fmt.Sprintf("plan %s@%d not found", e.Key, e.Version)
}
