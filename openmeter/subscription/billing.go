package subscription

type BillingBehaviorOverride struct {
	// If true, the billing cadence will be restarted
	// The anchor time will be the time the originating change takes effect,
	// which in practive translates to a SubscriptionItem's ActiveFrom property.
	RestartBillingPeriod *bool `json:"restartBillingPeriod,omitempty"`

	// ProratingBehavior will also be configurable here, but it's ignored for now
	// ProrateItem any
}
