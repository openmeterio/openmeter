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

type RateCard interface {
	ToCreateSubscriptionItemPlanInput() CreateSubscriptionItemPlanInput
	Key() string
}

type PlanPhase interface {
	ToCreateSubscriptionPhasePlanInput() CreateSubscriptionPhasePlanInput
	RateCards() []RateCard
	Key() string
}

type Plan interface {
	ToCreateSubscriptionPlanInput() CreateSubscriptionPlanInput
	Phases() []PlanPhase
	Key() string
	Version() int
}

type PlanNotFoundError struct {
	Key     string
	Version int
}

func (e *PlanNotFoundError) Error() string {
	return fmt.Sprintf("plan %s@%d not found", e.Key, e.Version)
}
