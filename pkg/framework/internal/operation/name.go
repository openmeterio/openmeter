package operation

import "context"

type contextKey string

const operationKey = contextKey("operation")

// Attach an operation name to the context.
func ContextWithName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, operationKey, name)
}

// Name returns the name of the operation from the context (if any).
func Name(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(operationKey).(string)

	return name, ok
}
