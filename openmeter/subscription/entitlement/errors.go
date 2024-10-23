package subscriptionentitlement

import (
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type NotFoundError struct {
	ID      string
	ItemRef subscription.SubscriptionItemRef
	At      time.Time
}

func (e *NotFoundError) Error() string {
	msg := "subscription entitlement"
	if e.ID != "" {
		msg = fmt.Sprintf("%s with id %s", msg, e.ID)
	}
	msg = msg + " not found"
	if e.ItemRef.SubscriptionId != "" {
		msg = fmt.Sprintf("%s for subscription %s", msg, e.ItemRef.SubscriptionId)
	}
	if e.ItemRef.PhaseKey != "" {
		msg = fmt.Sprintf("%s phase %s", msg, e.ItemRef.PhaseKey)
	}
	if e.ItemRef.ItemKey != "" {
		msg = fmt.Sprintf("%s item %s", msg, e.ItemRef.ItemKey)
	}
	if !e.At.IsZero() {
		msg = fmt.Sprintf("%s at %s", msg, e.At)
	}

	return msg
}

type AlreadyExistsError struct {
	ItemRef       subscription.SubscriptionItemRef
	EntitlementId string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("entitlement %s already exists for subscription %s phase %s item %s", e.EntitlementId, e.ItemRef.SubscriptionId, e.ItemRef.PhaseKey, e.ItemRef.ItemKey)
}
