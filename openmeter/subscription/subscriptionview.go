package subscription

import (
	"time"
)

type SubscriptionView interface {
	Sub() Subscription
	Phases() []SubscriptionPhaseView
}

type SubscriptionPhaseView interface {
	Key() string
	ActiveFrom() time.Time
	Items() []SubscriptionItemView
}

type SubscriptionItemView interface {
	BillingCadence() time.Duration
	Key() string

	PriceID() (string, bool)
	FeatureKey() (string, bool)
	EntitlementID() (string, bool)
}
