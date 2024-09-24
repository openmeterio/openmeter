package subscription

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

// Plan is a dummy representation that can be used internally
type Plan struct {
	models.NamespacedModel
	models.ManagedModel
	models.VersionedModel

	ID string `json:"id,omitempty"`
}

type PlanPhase interface {
	RateCards() []RateCard
	Duration() time.Duration

	UniquelyComparable
}

type RateCard interface {
	GetFeatureIdOrKey() (string, error)
	GetEntitlementSpec() (entitlement.CreateEntitlementInputs, error)

	UniquelyComparable
}

type DoesntHaveResourceError struct {
	ResourceName string
}

func (e *DoesntHaveResourceError) Error() string {
	return fmt.Sprintf("rate card doesn't have resource: %s", e.ResourceName)
}

type PlanAdapter interface {
	GetVersion(ctx context.Context, planRef modelref.VersionedKeyRef) (Plan, error)
	GetPhases(ctx context.Context, planRef modelref.VersionedKeyRef) ([]PlanPhase, error)
}
