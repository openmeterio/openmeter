package subscriptionentitlement

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionEntitlement struct {
	models.ManagedResource
	subscription.SubscriptionItemRef
	EntitlementId string `json:"entitlementId"`
}

type CreateSubscriptionEntitlementInput struct {
	Namespace           string
	EntitlementId       string
	SubscriptionItemRef subscription.SubscriptionItemRef
}

type Repository interface {
	Create(ctx context.Context, ent CreateSubscriptionEntitlementInput) (SubscriptionEntitlement, error)
	Get(ctx context.Context, id string) (SubscriptionEntitlement, error)
	GetBySubscriptionItem(ctx context.Context, ref subscription.SubscriptionItemRef, at time.Time) (SubscriptionEntitlement, error)
}
