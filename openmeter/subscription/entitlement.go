package subscription

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionEntitlement struct {
	Entitlement entitlement.Entitlement
	Cadence     models.CadencedModel
	ItemRef     SubscriptionItemRef
}

type EntitlementAdapter interface {
	ScheduleEntitlement(ctx context.Context, ref SubscriptionItemRef, input entitlement.CreateEntitlementInputs) (*SubscriptionEntitlement, error)
	// GetForItem is a bit tricky, as ItemRef is a stable reference while the underlying entitlemnet might change with edits/deletes.
	GetForItem(ctx context.Context, ref SubscriptionItemRef, at time.Time) (*SubscriptionEntitlement, error)
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]SubscriptionEntitlement, error)
}
