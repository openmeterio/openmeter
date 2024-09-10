package subscription

type Subscription struct {
	Namespace string `json:"-"`
	ID        string `json:"id,omitempty"`
	// dummy
}

func (s *Subscription) IsTrialing() bool {
	panic("not implemented")
}

type SubscriptionCreateInfo struct {
	// dummy
}
type SubscriptionOverrides struct {
	// dummy
}
