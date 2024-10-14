package subscription

import (
	"time"
)

type SubscriptionView interface {
	Sub() Subscription
	Phases() []SubscriptionPhaseView
}

type SubscriptionPhaseView interface {
	Key() PhaseKey
	ActiveFrom() time.Time
	Items() []SubscriptionItemView
}

type SubscriptionItemView interface {
	BillingCadence() time.Duration
	Key() ItemKey

	PriceID() (string, bool)
	FeatureKey() (string, bool)
	EntitlementID() (string, bool)
}
