package subscription

import (
	"context"
	"time"
)

type NewSubscriptionRequest struct {
	Namespace  string
	ActiveFrom time.Time
	CustomerID string

	Plan struct {
		Key     string
		Version int
	}

	// The SubscriptionItem customizations compared to the plan
	ItemCustomization []Patch

	// TODO: Add discounts, either separately or as part of the patch language
}

type Connector interface {
	Create(ctx context.Context, req NewSubscriptionRequest) (Subscription, error)
	Edit(ctx context.Context, subscriptionID string, patches []Patch) (Subscription, error)
	End(ctx context.Context, subscriptionID string, at time.Time) (Subscription, error)
}
