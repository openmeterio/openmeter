package subscriptionentitlement

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type NotFoundError struct {
	ItemRef subscription.SubscriptionItemRef
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("entitlement not found for subscription %s phase %s item %s", e.ItemRef.SubscriptionId, e.ItemRef.PhaseKey, e.ItemRef.ItemKey)
}

type AlreadyExistsError struct {
	ItemRef       subscription.SubscriptionItemRef
	EntitlementId string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement %s already exists for subscription %s phase %s item %s", e.EntitlementId, e.ItemRef.SubscriptionId, e.ItemRef.PhaseKey, e.ItemRef.ItemKey)
}
