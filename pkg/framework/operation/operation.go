// Package operation provides an abstraction for RPC-style APIs.
//
// Implementations are generally business logic functions that take a request and return a response.
// Consumers of operations are transport layers, such as HTTP handlers or gRPC services.
//
// The operation layer allows you to separate business logic from transport concerns as well as to compose and reuse transport and operation-agnostic logic.
package operation

import (
	"context"
)

// Operation is the fundamental building block of RPC-style APIs.
// It represents a single operation that can be performed by a caller.
type Operation[Request any, Response any] func(ctx context.Context, request Request) (Response, error)

// AsNoResponseOperation wraps a func (context.Context, request Request) error typed function as an operation
// useful for Delete like methods
func AsNoResponseOperation[Request any](f func(ctx context.Context, request Request) error) Operation[Request, any] {
	return func(ctx context.Context, request Request) (any, error) {
		return nil, f(ctx, request)
	}
}

// Compose can be used to chain two operations together (e.g. get something, then update it).
func Compose[Request any, Intermediate any, Response any](op1 Operation[Request, Intermediate], op2 Operation[Intermediate, Response]) Operation[Request, Response] {
	return func(ctx context.Context, request Request) (Response, error) {
		intermediate, err := op1(ctx, request)
		if err != nil {
			var defaultResponse Response
			return defaultResponse, err
		}
		return op2(ctx, intermediate)
	}
}
