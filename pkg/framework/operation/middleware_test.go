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
