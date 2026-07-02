package subscription

import "context"

type contextKey string

const (
	subscriptionoperation contextKey = "subscriptionoperation"
	grantoperation        contextKey = "grantoperation"
)

func NewSubscriptionOperationContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, subscriptionoperation, true)
}

func IsSubscriptionOperation(ctx context.Context) bool {
	u, ok := ctx.Value(subscriptionoperation).(bool)
	if !ok {
		return false
	}

	return u
}

// NewGrantOperationContext marks ctx as a grant creation operation.
// The subscription hook allows grants on subscription-managed entitlements
// because grants are ad-hoc credit top-ups that do not alter the entitlement
// structure managed by the subscription.
func NewGrantOperationContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, grantoperation, true)
}

// IsGrantOperation reports whether ctx was created by NewGrantOperationContext.
func IsGrantOperation(ctx context.Context) bool {
	u, ok := ctx.Value(grantoperation).(bool)
	return ok && u
}
