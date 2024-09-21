// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
