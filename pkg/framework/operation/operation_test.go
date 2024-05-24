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
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/framework/operation"
)

type ExampleRequest struct {
	Name string
}

type ExampleResponse struct {
	Greeting string
}

func exampleOperation(ctx context.Context, request ExampleRequest) (ExampleResponse, error) {
	if request.Name == "" {
		return ExampleResponse{}, errors.New("name is required")
	}

	return ExampleResponse{Greeting: "Hello, " + request.Name}, nil
}

func ExampleOperation() {
	var op operation.Operation[ExampleRequest, ExampleResponse] = exampleOperation

	resp, err := op(context.Background(), ExampleRequest{Name: "World"})
	if err != nil {
		panic(err)
	}

	fmt.Print(resp.Greeting)
	// Output: Hello, World
}

// func ExampleOperation() {
// 	var op operation.Operation[ExampleRequest, ExampleResponse] = operationImpl{}
//
// 	resp, err := op.Do(context.Background(), ExampleRequest{Name: "World"})
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Print(resp.Greeting)
// 	// Output: Hello, World
// }

// func ExampleOperationFunc() {
// 	var op operation.Operation[ExampleRequest, ExampleResponse] = operation.OperationFunc[ExampleRequest, ExampleResponse](exampleOperation)
//
// 	resp, err := op.Do(context.Background(), ExampleRequest{Name: "World"})
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Print(resp.Greeting)
// 	// Output: Hello, World
// }
//
// func ExampleNew() {
// 	op := operation.New(exampleOperation)
//
// 	resp, err := op.Do(context.Background(), ExampleRequest{Name: "World"})
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	fmt.Print(resp.Greeting)
// 	// Output: Hello, World
// }
