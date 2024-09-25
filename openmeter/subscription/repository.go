package subscription

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type SubscriptionRepo interface {
	// Create a new subscription.
	Create(ctx context.Context, subscription SubscriptionCreateInput) (Subscription, error)
	GetByID(ctx context.Context, subscriptionID string) (Subscription, error)

	UpdateCadence(ctx context.Context, subscriptionID string, cadence models.CadencedModel) (Subscription, error)
}

type SubscriptionPhaseRepo interface {
	Create(ctx context.Context, phase SubscriptionPhaseCreateInput) (SubscriptionPhase, error)
	DeleteAt(ctx context.Context, id string, at time.Time) error

	GetForSub(ctx context.Context, subscriptionID string) ([]SubscriptionPhase, error)
	GetRateCards(ctx context.Context, phaseID string) ([]SubscriptionRateCard, error)
}

type SubscriptionEntitlementRepo interface {
	GetByRateCard(ctx context.Context, rateCardID string) (Entitlement, error)
}

type Entitlement struct {
	Entitlement    entitlement.Entitlement
	RateCardID     string
	SubscriptionID string
}

// CustomerSubscriptionRepo is a repository for interacting with subscriptions in the context of a customer.
type CustomerSubscriptionRepo interface {
	// GetAll returns all subscriptions for a customer.
	// TODO: add pagination.
	GetAll(ctx context.Context, customerID string, params CustomerSubscriptionRepoParams) ([]Subscription, error)
}

type CustomerSubscriptionRepoParams struct {
	PlanKey *string `json:"planKey,omitempty"`
}

type SubscriptionCreateInput struct {
	models.NamespacedModel
	models.CadencedModel

	TemplatingPlanRef modelref.VersionedKeyRef
}

type SubscriptionPhaseCreateInput struct {
	models.NamespacedModel
	ActiveFrom     time.Time
	SubscriptionId string
}
