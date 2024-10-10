package subscription

import (
	"time"
)

type SubscriptionView interface{}

type SubscriptionPhaseView interface {
	Key() PhaseKey
	ActiveFrom() time.Time
}

type SubscriptionItemView interface {
	BillingCadence() time.Duration
	Key() ItemKey

	PriceID() (string, bool)
	FeatureKey() (string, bool)
	EntitlementID() (string, bool)
}
