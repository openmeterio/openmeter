package subscription

import "context"

type contextKey string

const (
	subscriptionoperation contextKey = "subscriptionoperation"
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
