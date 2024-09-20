package subscription

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type SubscriptionRepo interface {
	// Create a new subscription.
	Create(ctx context.Context, subscription SubscriptionCreateInput) (Subscription, error)
	GetByID(ctx context.Context, subscriptionID modelref.IDRef) (Subscription, error)

	UpdateCadence(ctx context.Context, subscriptionID modelref.IDRef, cadence models.CadencedModel) (Subscription, error)
}

type CustomerSubscriptionRepo interface {
	// GetActiveSubscriptionsAt returns the active subscriptions for a customer at the given time.
	//
	// Each customer can have multiple active subscriptions at a time, given that:
	// - At most one of them is trialing
	// - At most of them is non-trialing
	GetActiveSubscriptionsAt(ctx context.Context, customerID modelref.IDRef, at time.Time) ([]Subscription, error)

	// GetEffectiveAt returns the subscription that is effective at the given time.
	//
	// An effective subscription is an active subscription. If there are multiple active subscriptions
	// the trialing subscription is effective.
	GetEffectiveAt(ctx context.Context, customerID modelref.IDRef, at time.Time) (Subscription, error)

	// GetCurrentAt returns the subscription that is current at the given time.
	// The current subscription is a subscription that cannot be trialing.
	GetCurrentAt(ctx context.Context, customerID modelref.IDRef, at time.Time) (Subscription, error)

	GetAll(ctx context.Context, customerID modelref.IDRef, filters SubscriptionFilters) ([]Subscription, error)
}

type ContentRepo interface {
	// GetContentsAt returns the contents of a subscription at the given time.
	GetContentsAt(ctx context.Context, subscriptionRef modelref.IDOrKeyRef, at time.Time) ([]Content, error)

	// Create a new content for a subscription.
	CreateMany(ctx context.Context, subscriptionID modelref.IDRef, contents []ContentCreateInput) ([]Content, error)
}

type SubscriptionFilters struct {
	PlanKey *modelref.KeyRef `json:"planKey,omitempty"`
}
