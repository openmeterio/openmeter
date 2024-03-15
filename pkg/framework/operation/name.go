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

package operation

// Name returns the name of the operation from the context (if any).
// func Name(ctx context.Context) (string, bool) {
// 	return operation.Name(ctx)
// }

// WithName returns a new [Operation] attaching a name to the operation, which can be used for logging and debugging.
//
// This middleware should be the highest in the stack, so that transports can detect it properly.
//
// Operation names should be unique within a service.
// func WithName[Request any, Response any](name string, op Operation[Request, Response]) Operation[Request, Response] {
// 	return withName[Request, Response]{op: op, name: name}
// }
//
// type withName[Request any, Response any] struct {
// 	op   Operation[Request, Response]
// 	name string
// }
//
// func (o withName[Request, Response]) Do(ctx context.Context, request Request) (Response, error) {
// 	ctx = operation.ContextWithName(ctx, o.name)
//
// 	return o.op.Do(ctx, request)
// }
//
// func (o withName[Request, Response]) Name() string {
// 	return o.name
// }
