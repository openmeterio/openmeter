package subscriptiontestutils

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type mockBillingAdapter struct {
	store map[subscription.SubscriptionItemRef]time.Time
}

func (m *mockBillingAdapter) ItemLastInvoicedAt(_ context.Context, _ string, itemRef subscription.SubscriptionItemRef) (*time.Time, error) {
	if m.store == nil {
		m.store = make(map[subscription.SubscriptionItemRef]time.Time)
	}

	if t, ok := m.store[itemRef]; ok {
		return &t, nil
	}

	return nil, nil
}

func (m *mockBillingAdapter) Set(itemRef subscription.SubscriptionItemRef, t time.Time) {
	if m.store == nil {
		m.store = make(map[subscription.SubscriptionItemRef]time.Time)
	}

	m.store[itemRef] = t
}

func (m *mockBillingAdapter) Unset(itemRef subscription.SubscriptionItemRef) {
	delete(m.store, itemRef)
}

var _ subscription.BillingAdapter = &mockBillingAdapter{}

func NewMockBillingAdapter() *mockBillingAdapter {
	return &mockBillingAdapter{}
}
