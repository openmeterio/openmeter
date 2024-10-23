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
	// // At refers to a point in time for which we're querying the system state, meaning:
	// if t1 < t2 < t3, and some entitlement was deleted effective at t2, then
	// with at = t1 the entitlement will be returned, while with at = t3 it won't.
	//
	// As SubscriptionItemRef is a stable ref while the underlying entitlement might change,
	// logically changed entitlemnets have to be deleted.
	GetForItem(ctx context.Context, namespace string, ref SubscriptionItemRef, at time.Time) (*SubscriptionEntitlement, error)
	// At refers to a point in time for which we're querying the system state, meaning:
	// if t1 < t2 < t3, and some entitlement was deleted effective at t2, then
	// with at = t1 the entitlement will be returned, while with at = t3 it won't.
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]SubscriptionEntitlement, error)
}
