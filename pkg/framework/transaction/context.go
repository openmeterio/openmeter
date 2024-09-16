package transaction

import (
	"context"
)

type omTransactionContextKey string

const contextKey omTransactionContextKey = "om_transaction_context_key"

func GetDriverFromContext(ctx context.Context) (Driver, error) {
	tx, ok := ctx.Value(contextKey).(Driver)
	if !ok {
		return nil, &DriverNotFoundError{}
	}
	return tx, nil
}

type DriverNotFoundError struct{}

func (e *DriverNotFoundError) Error() string {
	return "tx driver not found in context"
}

func SetDriverOnContext(ctx context.Context, tx Driver) (context.Context, error) {
	if _, err := GetDriverFromContext(ctx); err == nil {
		return ctx, &DriverConflictError{}
	}
	return context.WithValue(ctx, contextKey, tx), nil
}

type DriverConflictError struct{}

func (e *DriverConflictError) Error() string {
	return "tx driver already exists in context"
}
