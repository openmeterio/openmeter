package subscription

import (
	"context"
	"fmt"
)

type PlanAdapter interface {
	// GetPlan returns the plan with the given key and version with all it's dependent resources.
	//
	// If the Plan is Not Found, it should return a PlanNotFoundError.
	GetVersion(ctx context.Context, planKey string, version int) (any, error)
}

type RateCard interface {
	ToSubscriptionItemCreateInput() CreateSubscriptionItemInput
}

type PlanPhase interface {
	ToSubscriptionPhaseCreateInput() CreateSubscriptionPhaseInput
	RateCards() []RateCard
}

type Plan interface {
	Phases() []PlanPhase
}

type PlanNotFoundError struct {
	Key     string
	Version int
}

func (e *PlanNotFoundError) Error() string {
	return fmt.Sprintf("plan %s@%d not found", e.Key, e.Version)
}
