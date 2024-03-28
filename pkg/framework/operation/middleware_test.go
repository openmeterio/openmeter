package operation_test

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

func mw1[Request any, Response any](next operation.Operation[Request, Response]) operation.Operation[Request, Response] {
	return func(ctx context.Context, request Request) (Response, error) {
		return next(ctx, request)
	}
}

func mwchain[Request any, Response any](op operation.Operation[Request, Response]) operation.Operation[Request, Response] {
	chain := operation.Chain[Request, Response](mw1)

	return chain(op)
}

func ExampleMiddleware() {
	mwchain(exampleOperation)
}
