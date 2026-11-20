package entitlement

import "context"

// CurrentValueOperation identifies calls that explicitly ask for a current
// value. An arbitrary evaluation time, even one close to now, is historical.
type CurrentValueOperation uint8

const (
	CurrentValueOperationRead CurrentValueOperation = iota + 1
	CurrentValueOperationCalculate
)

type currentValueOperationKey struct{}

func (o CurrentValueOperation) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, currentValueOperationKey{}, o)
}

func CurrentValueOperationFromContext(ctx context.Context) CurrentValueOperation {
	op, _ := ctx.Value(currentValueOperationKey{}).(CurrentValueOperation)
	return op
}
