package subscription

import (
	"context"
	"time"
)

type BillingAdapter interface {
	// Returns when the item was last invoiced at, or nil if its never been invoiced
	ItemLastInvoicedAt(ctx context.Context, namespace string, itemRef SubscriptionItemRef) (*time.Time, error)
}
