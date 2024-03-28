// Package operation provides an abstraction for RPC-style APIs.
//
// Implementations are generally business logic functions that take a request and return a response.
// Consumers of operations are transport layers, such as HTTP handlers or gRPC services.
//
// The operation layer allows you to separate business logic from transport concerns as well as to compose and reuse transport and operation-agnostic logic.
package operation

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/internal/operation"
)

// Operation is the fundamental building block of RPC-style APIs.
// It represents a single operation that can be performed by a caller.
type Operation[Request any, Response any] func(ctx context.Context, request Request) (Response, error)

// Name returns the name of the operation from the context (if any).
func Name(ctx context.Context) (string, bool) {
	return operation.Name(ctx)
}
