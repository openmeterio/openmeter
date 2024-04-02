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
